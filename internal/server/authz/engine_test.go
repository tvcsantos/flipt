package authz

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEngine_NewEngine(t *testing.T) {
	ctx := context.Background()
	engine, err := NewEngine(ctx)
	require.NoError(t, err)
	require.NotNil(t, engine)
}

func TestEngine_IsAllowed(t *testing.T) {
	var tests = []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "admin is allowed to write",
			input: `{
				"role": "admin",
				"action": "write",
				"namespace": "default",
				"permissions": {
					"default": {
						"admin": {
							"write": true
						}
					}
				}
			}`,
			expected: true,
		},
		{
			name: "editor is not allowed to write",
			input: `{
				"role": "editor",
				"action": "write",
				"namespace": "default",
				"permissions": {
					"default": {
						"admin": {
							"write": true
						}
					}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			engine, err := NewEngine(ctx)
			require.NoError(t, err)

			var input map[string]interface{}

			err = json.Unmarshal([]byte(tt.input), &input)
			require.NoError(t, err)

			allowed, err := engine.IsAllowed(ctx, input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, allowed)
		})
	}
}
