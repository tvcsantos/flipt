package fliptauthz

test_allow_admin_write {
    allow with input as {
        "role": "admin",
        "action": "write",
        "resource": "flags"
    }
}

test_allow_editor_write_flags {
    allow with input as {
        "role": "editor",
        "action": "write",
        "resource": "flags"
    }
}

test_deny_viewer_write_flags {
    not allow with input as {
        "role": "viewer",
        "action": "write",
        "resource": "flags"
    }
}