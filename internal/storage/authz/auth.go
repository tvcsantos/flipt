package authz

import (
	"context"

	"go.flipt.io/flipt/internal/storage"
	"go.flipt.io/flipt/rpc/flipt/auth"
)

// Store persists Authorization policies
type Store interface {
	ListNamespacePolicies(context.Context, *storage.ListRequest[ListAuthorizationPoliciesPredicate]) (storage.ResultSet[*auth.AuthorizationPolicy], error)
	CreateNamespacePolicy(context.Context, *auth.CreateAuthorizationPolicyRequest) (*auth.AuthorizationPolicy, error)
	DeleteNamespacePolicy(context.Context, *auth.DeleteAuthorizationPolicyRequest) error
}

type ListAuthorizationPoliciesPredicate struct {
	NamespaceKey *string
	RoleKey      *string
}
