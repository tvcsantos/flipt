package authz

import (
	"context"

	"github.com/open-policy-agent/opa/rego"
)

type Engine struct {
	query rego.PreparedEvalQuery
}

var defaultPolicy = `package authz

default allow = false

# Decision rule to determine if an action is allowed for a role in a namespace
allow {
	input.role != ""
	input.action != ""
	input.namespace != ""
	input.permissions != null

	# Extract the role, action, and namespace from the input
	role := input.role
	action := input.action
	namespace := input.namespace

	# Get the permissions for the role in the specified namespace from the input
	permissions := input.permissions[namespace][role]

	# Check if the role has the permission to perform the action in the namespace
	can_perform_action(permissions, action)
}

# Helper function to check if the action is in the permissions list
can_perform_action(permissions, action) {
	permissions[action]
}
`

func NewEngine(ctx context.Context) (*Engine, error) {
	r := rego.New(
		rego.Query("data.authz.allow"),
		rego.Module("policy.rego", defaultPolicy),
	)

	query, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, err
	}

	return &Engine{query: query}, nil
}

func (e *Engine) IsAllowed(ctx context.Context, input map[string]interface{}) (bool, error) {
	results, err := e.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, err
	}

	if len(results) == 0 {
		return false, nil
	}

	return results[0].Expressions[0].Value.(bool), nil
}
