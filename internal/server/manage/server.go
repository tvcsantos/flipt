package manage

import (
	"context"
	"fmt"

	"go.flipt.io/flipt/internal/storage"
	"go.flipt.io/flipt/rpc/flipt/manage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Store interface {
	Update(ctx context.Context, ref storage.Reference, message string, flag *manage.Flag) (string, error)
}

type Server struct {
	logger *zap.Logger

	store Store

	manage.UnimplementedManageServiceServer
}

func NewServer(logger *zap.Logger, store Store) *Server {
	return &Server{logger: logger, store: store}
}

// RegisterGRPC registers the server as an Server on the provided grpc server.
func (s *Server) RegisterGRPC(server *grpc.Server) {
	manage.RegisterManageServiceServer(server, s)
}

func (s *Server) GetNamespace(_ context.Context, _ *manage.GetNamespaceRequest) (*manage.Namespace, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Server) PutFlag(ctx context.Context, flag *manage.Flag) (*manage.Proposal, error) {
	branch, err := s.store.Update(ctx, storage.Reference(""), fmt.Sprintf("feat: put flag %s", flag.Key), flag)
	if err != nil {
		return nil, err
	}

	return &manage.Proposal{Id: branch}, nil
}

func (s *Server) PutSegment(_ context.Context, _ *manage.Segment) (*manage.Proposal, error) {
	panic("not implemented") // TODO: Implement
}
