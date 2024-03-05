package authz_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.flipt.io/flipt/internal/server/authz"
)

func TestNewEnforcer(t *testing.T) {
	enforcer, err := authz.NewEnforcer()
	require.NoError(t, err)
	require.NotNil(t, enforcer)
}

func TestPolicies(t *testing.T) {
	enforcer, err := authz.NewEnforcer()
	require.NoError(t, err)

	allowed, err := enforcer.Enforce("admin", "*", "flags", "write")
	require.NoError(t, err)
	assert.True(t, allowed)
}
