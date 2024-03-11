package sql

import (
	"context"
	"fmt"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"go.flipt.io/flipt/internal/storage"
	storageauth "go.flipt.io/flipt/internal/storage/authz"
	storagesql "go.flipt.io/flipt/internal/storage/sql"
	rpcauth "go.flipt.io/flipt/rpc/flipt/auth"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Store struct {
	logger  *zap.Logger
	driver  storagesql.Driver
	builder sq.StatementBuilderType

	now func() *timestamppb.Timestamp

	generateID func() string
}

// Option is a type which configures a *Store
type Option func(*Store)

// NewStore constructs and configures a new instance of *Store.
// Queries are issued to the database via the provided statement builder.
func NewStore(driver storagesql.Driver, builder sq.StatementBuilderType, logger *zap.Logger, opts ...Option) *Store {
	store := &Store{
		logger:  logger,
		driver:  driver,
		builder: builder,
		now: func() *timestamppb.Timestamp {
			// we truncate timestamps to the microsecond to support Postgres/MySQL
			// the lowest common denominators in terms of timestamp precision
			now := time.Now().UTC().Truncate(time.Microsecond)
			return timestamppb.New(now)
		},
		generateID: func() string {
			return uuid.Must(uuid.NewV4()).String()
		},
	}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// WithNowFunc overrides the stores now() function used to obtain
// a protobuf timestamp representative of the current time of evaluation.
func WithNowFunc(fn func() *timestamppb.Timestamp) Option {
	return func(s *Store) {
		s.now = fn
	}
}

// WithIDGeneratorFunc overrides the stores ID generator function
// used to generate new random ID strings, when creating new instances
// of AuthorizationPolicies.
// The default is a string containing a valid UUID (V4).
func WithIDGeneratorFunc(fn func() string) Option {
	return func(s *Store) {
		s.generateID = fn
	}
}

func (s *Store) CreateNamespacePolicy(ctx context.Context, req *rpcauth.CreateAuthorizationPolicyRequest) (*rpcauth.AuthorizationPolicy, error) {
	var (
		authorization = rpcauth.AuthorizationPolicy{
			Id:           s.generateID(),
			NamespaceKey: req.NamespaceKey,
			RoleKey:      req.RoleKey,
			Action:       req.Action,
		}
	)

	if _, err := s.builder.Insert("namespace_policies").
		Columns(
			"id",
			"namespace_key",
			"role_key",
			"action",
			"created_at",
			"updated_at",
		).Values(
		&authorization.Id,
		&authorization.NamespaceKey,
		&authorization.RoleKey,
		&authorization.Action,
		s.now(),
		s.now(),
	).ExecContext(ctx); err != nil {
		return nil, fmt.Errorf("inserting authorization policy: %w", s.driver.AdaptError(err))
	}

	return &authorization, nil
}

func (s *Store) ListNamespacePolicies(ctx context.Context, req *storage.ListRequest[storageauth.ListAuthorizationPoliciesPredicate]) (set storage.ResultSet[*rpcauth.AuthorizationPolicy], err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"listing authorization policies: %w",
				s.driver.AdaptError(err),
			)
		}
	}()

	// adjust the query parameters within normal bounds
	req.QueryParams.Normalize()

	query := s.builder.Select(
		"id",
		"namespace_key",
		"role_key",
		"action",
		"created_at",
		"updated_at",
	).From("namespace_policies").
		Limit(req.QueryParams.Limit+1).
		OrderBy(fmt.Sprintf("created_at %s", req.QueryParams.Order)).
		GroupBy("namespace_key", "role_key", "action")

	if req.Predicate.NamespaceKey != nil {
		query = query.Where(sq.Eq{"namespace_key": *req.Predicate.NamespaceKey})
	}

	var offset int
	if v, err := strconv.ParseInt(req.QueryParams.PageToken, 10, 64); err == nil {
		offset = int(v)
		query = query.Offset(uint64(v))
	}

	rows, err := query.QueryContext(ctx)
	if err != nil {
		return set, err
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var authorization rpcauth.AuthorizationPolicy
		if err = s.scanAuthorizationPolicy(rows, &authorization); err != nil {
			return
		}

		if len(set.Results) >= int(req.QueryParams.Limit) {
			// set the next page token to the first
			// row beyond the query limit and break
			set.NextPageToken = fmt.Sprintf("%d", offset+int(req.QueryParams.Limit))
			break
		}

		set.Results = append(set.Results, &authorization)
	}

	if err = rows.Err(); err != nil {
		return
	}

	return
}

func (s *Store) DeleteNamespacePolicy(ctx context.Context, req *rpcauth.DeleteAuthorizationPolicyRequest) error {
	_, err := s.builder.Delete("namespace_policies").
		Where(sq.Eq{"id": req.Id}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("deleting authorization policy: %w", s.driver.AdaptError(err))
	}

	return nil
}

func (s *Store) scanAuthorizationPolicy(scanner sq.RowScanner, authorizationPolicy *rpcauth.AuthorizationPolicy) error {
	var (
		createdAt storagesql.Timestamp
		updatedAt storagesql.Timestamp
	)

	if err := scanner.Scan(
		&authorizationPolicy.Id,
		&authorizationPolicy.NamespaceKey,
		&authorizationPolicy.RoleKey,
		&authorizationPolicy.Action,
		&createdAt,
		&updatedAt,
	); err != nil {
		return fmt.Errorf("reading authorization policy: %w", s.driver.AdaptError(err))
	}

	authorizationPolicy.CreatedAt = createdAt.Timestamp
	authorizationPolicy.UpdatedAt = updatedAt.Timestamp

	return nil
}
