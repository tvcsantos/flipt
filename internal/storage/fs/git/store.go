package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/gofrs/uuid"
	"go.flipt.io/flipt/internal/containers"
	"go.flipt.io/flipt/internal/ext"
	"go.flipt.io/flipt/internal/gitfs"
	"go.flipt.io/flipt/internal/storage"
	storagefs "go.flipt.io/flipt/internal/storage/fs"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// REFERENCE_CACHE_EXTRA_CAPACITY is the additionally capacity reserved in the cache
// for non-default references
const REFERENCE_CACHE_EXTRA_CAPACITY = 3

// ensure that the git *Store implements storage.ReferencedSnapshotStore
var _ storagefs.ReferencedSnapshotStore = (*SnapshotStore)(nil)

// SnapshotStore is an implementation of storage.SnapshotStore
// This implementation is backed by a Git repository and it tracks an upstream reference.
// When subscribing to this source, the upstream reference is tracked
// by polling the upstream on a configurable interval.
type SnapshotStore struct {
	*storagefs.Poller

	logger          *zap.Logger
	url             string
	baseRef         string
	auth            transport.AuthMethod
	insecureSkipTLS bool
	caBundle        []byte
	pollOpts        []containers.Option[storagefs.Poller]

	mu      sync.RWMutex
	repo    *git.Repository
	storage *memory.Storage

	snaps *storagefs.SnapshotCache[plumbing.Hash]
}

// WithRef configures the target reference to be used when fetching
// and building fs.FS implementations.
// If it is a valid hash, then the fixed SHA value is used.
// Otherwise, it is treated as a reference in the origin upstream.
func WithRef(ref string) containers.Option[SnapshotStore] {
	return func(s *SnapshotStore) {
		s.baseRef = ref
	}
}

// WithPollOptions configures the poller used to trigger update procedures
func WithPollOptions(opts ...containers.Option[storagefs.Poller]) containers.Option[SnapshotStore] {
	return func(s *SnapshotStore) {
		s.pollOpts = append(s.pollOpts, opts...)
	}
}

// WithAuth returns an option which configures the auth method used
// by the provided source.
func WithAuth(auth transport.AuthMethod) containers.Option[SnapshotStore] {
	return func(s *SnapshotStore) {
		s.auth = auth
	}
}

// WithInsecureTLS returns an option which configures the insecure TLS
// setting for the provided source.
func WithInsecureTLS(insecureSkipTLS bool) containers.Option[SnapshotStore] {
	return func(s *SnapshotStore) {
		s.insecureSkipTLS = insecureSkipTLS
	}
}

// WithCABundle returns an option which configures the CA Bundle used for
// validating the TLS connection to the provided source.
func WithCABundle(caCertBytes []byte) containers.Option[SnapshotStore] {
	return func(s *SnapshotStore) {
		if caCertBytes != nil {
			s.caBundle = caCertBytes
		}
	}
}

// NewSnapshotStore constructs and configures a Store.
// The store uses the connection and credential details provided to build
// fs.FS implementations around a target git repository.
func NewSnapshotStore(ctx context.Context, logger *zap.Logger, url string, opts ...containers.Option[SnapshotStore]) (_ *SnapshotStore, err error) {
	store := &SnapshotStore{
		logger:  logger.With(zap.String("repository", url)),
		url:     url,
		baseRef: "main",
	}
	containers.ApplyAll(store, opts...)

	store.logger = store.logger.With(zap.String("ref", store.baseRef))

	store.snaps, err = storagefs.NewSnapshotCache[plumbing.Hash](logger, REFERENCE_CACHE_EXTRA_CAPACITY)
	if err != nil {
		return nil, err
	}

	store.storage = memory.NewStorage()
	store.repo, err = git.Clone(store.storage, nil, &git.CloneOptions{
		Auth:            store.auth,
		URL:             store.url,
		CABundle:        store.caBundle,
		InsecureSkipTLS: store.insecureSkipTLS,
	})
	if err != nil {
		return nil, err
	}

	// do an initial fetch to setup remote tracking branches
	if err := store.fetch(ctx); err != nil {
		return nil, err
	}

	// fetch base ref snapshot at-least once before returning store
	// to ensure we have a servable default state
	snap, hash, err := store.buildReference(ctx, store.baseRef)
	if err != nil {
		return nil, err
	}

	// base reference is stored as fixed in the cache
	// meaning the reference will never be evicted and
	// always point to a live snapshot
	store.snaps.AddFixed(ctx, store.baseRef, hash, snap)

	store.Poller = storagefs.NewPoller(store.logger, ctx, store.update, store.pollOpts...)

	go store.Poll()

	return store, nil
}

// String returns an identifier string for the store type.
func (*SnapshotStore) String() string {
	return "git"
}

// View accepts a function which takes a *StoreSnapshot.
// It supplies the provided function with a *Snapshot if one can be resolved for the requested revision reference.
// Providing an empty reference defaults View to using the stores base reference.
// The base reference will always be quickly accessible via minimal locking (single read-lock).
// Alternative references which have not yet been observed will be resolved and newly built into snapshots on demand.
func (s *SnapshotStore) View(ctx context.Context, storeRef storage.Reference, fn func(storage.ReadOnlyStore) error) error {
	ref := string(storeRef)
	if ref == "" {
		ref = s.baseRef
	}

	snap, ok := s.snaps.Get(ref)
	if ok {
		return fn(snap)
	}

	// force attempt a fetch to get the latest references
	if err := s.fetch(ctx); err != nil {
		return err
	}

	hash, err := s.resolve(ref)
	if err != nil {
		return err
	}

	snap, err = s.snaps.AddOrBuild(ctx, ref, hash, s.buildSnapshot)
	if err != nil {
		return err
	}

	return fn(snap)
}

// update fetches from the remote and given that a the target reference
// HEAD updates to a new revision, it builds a snapshot and updates it
// on the store.
func (s *SnapshotStore) update(ctx context.Context) (updated bool, _ error) {
	if err := s.fetch(ctx); err != nil {
		// either nothing updated or err != nil
		return false, err
	}

	var errs []error
	for _, ref := range s.snaps.References() {
		hash, err := s.resolve(ref)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		fn := func(ctx context.Context, hash plumbing.Hash) (*storagefs.Snapshot, error) {
			updated = true
			return s.buildSnapshot(ctx, hash)
		}
		if _, err := s.snaps.AddOrBuild(ctx, ref, hash, fn); err != nil {
			errs = append(errs, err)
		}
	}

	return updated, errors.Join(errs...)
}

func (s *SnapshotStore) fetch(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.repo.FetchContext(ctx, &git.FetchOptions{
		Auth: s.auth,
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/heads/*",
		},
	}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}

	return nil
}

func (s *SnapshotStore) buildReference(ctx context.Context, ref string) (*storagefs.Snapshot, plumbing.Hash, error) {
	hash, err := s.resolve(ref)
	if err != nil {
		return nil, plumbing.ZeroHash, err
	}

	snap, err := s.buildSnapshot(ctx, hash)
	if err != nil {
		return nil, plumbing.ZeroHash, err
	}

	return snap, hash, nil
}

func (s *SnapshotStore) Update(ctx context.Context, storeRef storage.Reference, namespace, message string, fn func(*ext.Document) error) (string, error) {
	var (
		create   bool
		ref      = string(storeRef)
		resolved = ref
	)

	if ref == "" {
		// if the supplied reference is empty we will attempt
		// to create a random one based on the base reference
		create = true
		ref = fmt.Sprintf("flipt/%s", uuid.Must(uuid.NewV4()))
		resolved = s.baseRef
	}

	hash, err := s.resolve(resolved)
	if err != nil {
		// return error if it was the base reference
		if resolved == s.baseRef {
			return "", err
		}

		// otherwise, attempt to create a new reference
		// using the one supplied
		s.logger.Debug("error attempting to resolve reference",
			zap.String("reference", resolved),
			zap.Error(err))

		create = true
	}

	// shallow copy the store without the existing index
	store := &memory.Storage{
		ReferenceStorage: s.storage.ReferenceStorage,
		ConfigStorage:    s.storage.ConfigStorage,
		ShallowStorage:   s.storage.ShallowStorage,
		ObjectStorage:    s.storage.ObjectStorage,
		ModuleStorage:    s.storage.ModuleStorage,
	}

	dir, err := os.MkdirTemp("", "flipt-update-*")
	if err != nil {
		return "", err
	}

	defer func() {
		_ = os.RemoveAll(dir)
	}()

	// open repository on store with in-memory workspace
	repo, err := git.Open(store, osfs.New(dir))
	if err != nil {
		return "", fmt.Errorf("open repo: %w", err)
	}

	work, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("open worktree: %w", err)
	}

	// create branch if reference is not known
	if create {
		if err := repo.CreateBranch(&config.Branch{
			Name:   ref,
			Remote: "origin",
		}); err != nil {
			return "", fmt.Errorf("create branch: %w", err)
		}

		if err := work.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(ref),
			Create: true,
			Hash:   hash,
		}); err != nil {
			return "", fmt.Errorf("checkout branch: %w", err)
		}
	} else {
		if err := work.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(ref),
		}); err != nil {
			return "", fmt.Errorf("checkout branch: %w", err)
		}
	}

	var (
		path string
		docs []*ext.Document
	)

	if err := storagefs.WalkDocuments(s.logger, os.DirFS(dir), func(name string, ds []*ext.Document) error {
		// search through documents in each file looking for a matching namespaced
		for _, doc := range ds {
			if doc.Namespace != namespace {
				continue
			}

			docs = ds
			path = name

			return fn(doc)
		}

		return nil
	}); err != nil {
		return "", err
	}

	if path == "" {
		return "", fmt.Errorf("namespace %q: not found", namespace)
	}

	fi, err := work.Filesystem.OpenFile(path, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("namespace %q: %w", namespace, err)
	}

	var encode func(any) error
	extn := filepath.Ext(path)
	switch extn {
	case ".yaml", ".yml":
		encode = yaml.NewEncoder(fi).Encode
	case "", ".json":
		encode = json.NewEncoder(fi).Encode
	default:
		_ = fi.Close()
		return "", fmt.Errorf("unexpected extension: %q", extn)
	}

	for _, doc := range docs {
		if err := encode(doc); err != nil {
			_ = fi.Close()
			return "", err
		}
	}

	if err := fi.Close(); err != nil {
		return "", err
	}

	// TODO(georgemac): report upstream to go-git
	// For some reason, even with AllowingEmptyCommits == false this
	// still produces empty commits and pushes them.
	// It could be to do with how the index is nuked before hand
	// to support concurrent requests (needs investigation).
	// For now lets just return empty when the status has nothing in it.
	status, err := work.Status()
	if err != nil {
		return "", fmt.Errorf("getting status: %w", err)
	}

	if len(status) == 0 {
		return "", nil
	}

	if err := work.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return "", fmt.Errorf("adding changes: %w", err)
	}

	var (
		now       = time.Now().UTC()
		signature = &object.Signature{
			Email: "dev@flipt.io",
			Name:  "dev",
			When:  now,
		}
	)
	_, err = work.Commit(message, &git.CommitOptions{
		Author:    signature,
		Committer: signature,
	})
	if err != nil {
		// NOTE: currently with go-git we can see https://github.com/go-git/go-git/issues/723
		// This occurs when the result of the removal leads to an empty repository.
		// Just an FYI why a delete might fail silently when the result is the target repo is empty.
		if errors.Is(err, git.ErrEmptyCommit) {
			return "", nil
		}

		return "", fmt.Errorf("committing changes: %w", err)
	}

	s.logger.Debug("Pushing Changes", zap.String("reference", ref))

	// push to proposed branch
	if err := repo.PushContext(ctx, &git.PushOptions{
		Auth:       s.auth,
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("%s:refs/heads/%s", ref, ref)),
			config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", ref, ref)),
		},
	}); err != nil {
		return "", fmt.Errorf("pushing changes: %w", err)
	}

	return ref, nil
}

func (s *SnapshotStore) resolve(ref string) (plumbing.Hash, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if plumbing.IsHash(ref) {
		return plumbing.NewHash(ref), nil
	}

	reference, err := s.repo.Reference(plumbing.NewBranchReferenceName(ref), true)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	return reference.Hash(), nil
}

// buildSnapshot builds a new store snapshot based on the provided hash.
func (s *SnapshotStore) buildSnapshot(ctx context.Context, hash plumbing.Hash) (*storagefs.Snapshot, error) {
	fs, err := gitfs.NewFromRepoHash(s.logger, s.repo, hash)
	if err != nil {
		return nil, err
	}

	return storagefs.SnapshotFromFS(s.logger, fs)
}
