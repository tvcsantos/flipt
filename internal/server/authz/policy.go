package authz

import "github.com/casbin/casbin/v2"

func NewEnforcer() (*casbin.Enforcer, error) {
	return casbin.NewEnforcer("policies/rbac_model.conf", "policies/rbac_policy.csv")
}
