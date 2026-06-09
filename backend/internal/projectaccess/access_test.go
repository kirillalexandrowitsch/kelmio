package projectaccess

import "testing"

func TestPermissionsForRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		workspaceRole    string
		projectRole      string
		isWorkspaceAdmin bool
		canRead          bool
		canManage        bool
	}{
		{name: "workspace admin override", workspaceRole: "admin", isWorkspaceAdmin: true, canRead: true, canManage: true},
		{name: "lead", workspaceRole: "member", projectRole: "lead", canRead: true, canManage: true},
		{name: "contributor", workspaceRole: "member", projectRole: "contributor", canRead: true},
		{name: "viewer", workspaceRole: "member", projectRole: "viewer", canRead: true},
		{name: "no membership", workspaceRole: "member"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			isWorkspaceAdmin, canRead, canManage := permissionsForRoles(tt.workspaceRole, tt.projectRole)
			if isWorkspaceAdmin != tt.isWorkspaceAdmin || canRead != tt.canRead || canManage != tt.canManage {
				t.Fatalf(
					"permissionsForRoles(%q, %q) = %v/%v/%v, want %v/%v/%v",
					tt.workspaceRole,
					tt.projectRole,
					isWorkspaceAdmin,
					canRead,
					canManage,
					tt.isWorkspaceAdmin,
					tt.canRead,
					tt.canManage,
				)
			}
		})
	}
}
