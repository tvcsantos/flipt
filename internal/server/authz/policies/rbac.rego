package fliptauthz

import rego.v1

default allow = false

permissions := {
    "admin": {"read": ["*"], "write": ["*"]},
    "viewer": {"read": ["*"]},
    "editor": {"read": ["*"], "write": ["flags", "segments"]}
}

can_perform_action(role, action, resource) if {
    allowed_resources := permissions[role][action]
    allowed_resources[_] == resource
}

can_perform_action(role, action, resource) if {
    allowed_resources := permissions[role][action]
    allowed_resources[_] == "*"
}

allow if {
    input.role != ""                            # Ensure the input includes a role
    action := input.action                      # Action requested by the user
    resource := input.resource                  # Resource user is attempting to access
    can_perform_action(input.role, action, resource)
}