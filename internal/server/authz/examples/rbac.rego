package authz

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
