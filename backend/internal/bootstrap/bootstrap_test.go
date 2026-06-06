package bootstrap

import (
	"strings"
	"testing"
)

func TestNormalizeConfig(t *testing.T) {
	t.Parallel()

	got, err := NormalizeConfig(Config{
		WorkspaceName:    "  Production   Workspace ",
		AdminEmail:       " Admin@Example.COM ",
		AdminUsername:    " Admin_User ",
		AdminDisplayName: " Production   Admin ",
		AdminPassword:    " secure-password ",
	})
	if err != nil {
		t.Fatalf("NormalizeConfig() error = %v", err)
	}

	if got.WorkspaceName != "Production Workspace" {
		t.Fatalf("WorkspaceName = %q", got.WorkspaceName)
	}
	if got.AdminEmail != "admin@example.com" {
		t.Fatalf("AdminEmail = %q", got.AdminEmail)
	}
	if got.AdminUsername != "admin_user" {
		t.Fatalf("AdminUsername = %q", got.AdminUsername)
	}
	if got.AdminDisplayName != "Production Admin" {
		t.Fatalf("AdminDisplayName = %q", got.AdminDisplayName)
	}
	if got.AdminPassword != "secure-password" {
		t.Fatalf("AdminPassword = %q", got.AdminPassword)
	}
}

func TestNormalizeConfigValidation(t *testing.T) {
	t.Parallel()

	valid := Config{
		WorkspaceName:    "Production Workspace",
		AdminEmail:       "admin@example.com",
		AdminUsername:    "admin_user",
		AdminDisplayName: "Production Admin",
		AdminPassword:    "secure-password",
	}

	tests := []struct {
		name   string
		update func(*Config)
		want   string
	}{
		{name: "missing workspace", update: func(cfg *Config) { cfg.WorkspaceName = "" }, want: "BOOTSTRAP_WORKSPACE_NAME is required"},
		{name: "long workspace", update: func(cfg *Config) { cfg.WorkspaceName = strings.Repeat("w", 81) }, want: "BOOTSTRAP_WORKSPACE_NAME must be 80 characters or fewer"},
		{name: "invalid email", update: func(cfg *Config) { cfg.AdminEmail = "not-email" }, want: "BOOTSTRAP_ADMIN_EMAIL is invalid"},
		{name: "invalid username", update: func(cfg *Config) { cfg.AdminUsername = "NO" }, want: "BOOTSTRAP_ADMIN_USERNAME must be 3-32 characters"},
		{name: "missing display name", update: func(cfg *Config) { cfg.AdminDisplayName = "" }, want: "BOOTSTRAP_ADMIN_DISPLAY_NAME is required"},
		{name: "long display name", update: func(cfg *Config) { cfg.AdminDisplayName = strings.Repeat("n", 81) }, want: "BOOTSTRAP_ADMIN_DISPLAY_NAME must be 80 characters or fewer"},
		{name: "short password", update: func(cfg *Config) { cfg.AdminPassword = "short" }, want: "BOOTSTRAP_ADMIN_PASSWORD must be at least 8 characters"},
		{name: "long password", update: func(cfg *Config) { cfg.AdminPassword = strings.Repeat("p", 129) }, want: "BOOTSTRAP_ADMIN_PASSWORD must be 128 characters or fewer"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := valid
			tt.update(&cfg)

			_, err := NormalizeConfig(cfg)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("NormalizeConfig() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestLoadConfigFromEnvRequiresAllValues(t *testing.T) {
	for _, key := range []string{
		"BOOTSTRAP_WORKSPACE_NAME",
		"BOOTSTRAP_ADMIN_EMAIL",
		"BOOTSTRAP_ADMIN_USERNAME",
		"BOOTSTRAP_ADMIN_DISPLAY_NAME",
		"BOOTSTRAP_ADMIN_PASSWORD",
	} {
		t.Setenv(key, "")
	}

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatal("LoadConfigFromEnv() expected error")
	}
}
