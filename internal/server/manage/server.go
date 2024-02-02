package manage

import (
	"context"
	"fmt"

	"go.flipt.io/flipt/internal/ext"
	"go.flipt.io/flipt/internal/storage"
	"go.flipt.io/flipt/rpc/flipt/manage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Store interface {
	Update(ctx context.Context, ref storage.Reference, namespace, message string, fn func(*ext.Document) error) (string, error)
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
	branch, err := s.store.Update(
		ctx,
		storage.Reference(""),
		flag.Namespace,
		fmt.Sprintf("feat: put flag %s/%s", flag.Namespace, flag.Key),
		func(doc *ext.Document) error {
			newFlag := &ext.Flag{
				Type:        flag.Type.String(),
				Key:         flag.Key,
				Name:        flag.Name,
				Description: flag.Description,
				Enabled:     flag.Enabled,
				Variants:    make([]*ext.Variant, 0, len(flag.Variants)),
				Rules:       make([]*ext.Rule, 0, len(flag.Rules)),
				Rollouts:    make([]*ext.Rollout, 0, len(flag.Rollouts)),
			}

			for _, variant := range flag.Variants {
				newFlag.Variants = append(newFlag.Variants, &ext.Variant{
					Key:         variant.Key,
					Name:        variant.Name,
					Description: variant.Description,
					Attachment:  variant.Attachment,
				})
			}

			for i, rule := range flag.Rules {
				newRule := &ext.Rule{
					Rank: uint(i + 1),
					Segment: &ext.SegmentEmbed{
						IsSegment: &ext.Segments{
							Keys:            rule.Segments,
							SegmentOperator: rule.SegmentOperator.String(),
						},
					},
					Distributions: make([]*ext.Distribution, 0, len(rule.Distributions)),
				}

				for _, dist := range rule.Distributions {
					newRule.Distributions = append(newRule.Distributions, &ext.Distribution{
						Rollout:    dist.Rollout,
						VariantKey: dist.Variant,
					})
				}

				newFlag.Rules = append(newFlag.Rules, newRule)
			}

			for _, rollout := range flag.Rollouts {
				newRollout := &ext.Rollout{
					Description: rollout.Description,
				}

				if segment := rollout.GetSegment(); segment != nil {
					newRollout.Segment = &ext.SegmentRule{
						Keys:     segment.Segments,
						Operator: segment.SegmentOperator.String(),
						Value:    segment.Value,
					}
				}

				if threshold := rollout.GetThreshold(); threshold != nil {
					newRollout.Threshold = &ext.ThresholdRule{
						Percentage: threshold.Percentage,
						Value:      threshold.Value,
					}
				}

				newFlag.Rollouts = append(newFlag.Rollouts, newRollout)
			}

			var found bool
			for i, f := range doc.Flags {
				if found = f.Key == flag.Key; found {
					doc.Flags[i] = newFlag
					break
				}
			}

			if !found {
				doc.Flags = append(doc.Flags, newFlag)
			}

			return nil
		})
	if err != nil {
		return nil, err
	}

	return &manage.Proposal{Id: branch}, nil
}

func (s *Server) PutSegment(ctx context.Context, segment *manage.Segment) (*manage.Proposal, error) {
	branch, err := s.store.Update(
		ctx,
		storage.Reference(""),
		segment.Namespace,
		fmt.Sprintf("feat: put segment %s/%s", segment.Namespace, segment.Key),
		func(doc *ext.Document) error {
			newSegment := &ext.Segment{
				MatchType:   segment.MatchType.String(),
				Key:         segment.Key,
				Name:        segment.Name,
				Description: segment.Description,
				Constraints: make([]*ext.Constraint, 0, len(segment.Constraints)),
			}

			for _, constraint := range segment.Constraints {
				newConstraint := &ext.Constraint{
					Type:        constraint.Type.String(),
					Description: constraint.Description,
					Operator:    constraint.Operator,
					Property:    constraint.Property,
					Value:       constraint.Value,
				}

				newSegment.Constraints = append(newSegment.Constraints, newConstraint)
			}

			var found bool
			for i, s := range doc.Segments {
				if found = s.Key == segment.Key; found {
					doc.Segments[i] = newSegment
					break
				}
			}

			if !found {
				doc.Segments = append(doc.Segments, newSegment)
			}

			return nil
		})
	if err != nil {
		return nil, err
	}

	return &manage.Proposal{Id: branch}, nil
}
