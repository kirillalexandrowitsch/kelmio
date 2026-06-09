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
		canWrite         bool
		canManage        bool
	}{
		{name: "workspace admin override", workspaceRole: "admin", isWorkspaceAdmin: true, canRead: true, canWrite: true, canManage: true},
		{name: "lead", workspaceRole: "member", projectRole: "lead", canRead: true, canWrite: true, canManage: true},
		{name: "contributor", workspaceRole: "member", projectRole: "contributor", canRead: true, canWrite: true},
		{name: "viewer", workspaceRole: "member", projectRole: "viewer", canRead: true},
		{name: "no membership", workspaceRole: "member"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			isWorkspaceAdmin, canRead, canWrite, canManage := permissionsForRoles(tt.workspaceRole, tt.projectRole)
			if isWorkspaceAdmin != tt.isWorkspaceAdmin || canRead != tt.canRead || canWrite != tt.canWrite || canManage != tt.canManage {
				t.Fatalf(
					"permissionsForRoles(%q, %q) = %v/%v/%v/%v, want %v/%v/%v/%v",
					tt.workspaceRole,
					tt.projectRole,
					isWorkspaceAdmin,
					canRead,
					canWrite,
					canManage,
					tt.isWorkspaceAdmin,
					tt.canRead,
					tt.canWrite,
					tt.canManage,
				)
			}
		})
	}
}
