package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	emailPattern    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,31}$`)

	ErrDatabaseNotEmpty = errors.New("bootstrap requires empty workspaces, users, and workspace_members tables")
)

type Config struct {
	WorkspaceName    string
	AdminEmail       string
	AdminUsername    string
	AdminDisplayName string
	AdminPassword    string
}

type Result struct {
	WorkspaceID string
	AdminUserID string
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func LoadConfigFromEnv() (Config, error) {
	return NormalizeConfig(Config{
		WorkspaceName:    os.Getenv("BOOTSTRAP_WORKSPACE_NAME"),
		AdminEmail:       os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
		AdminUsername:    os.Getenv("BOOTSTRAP_ADMIN_USERNAME"),
		AdminDisplayName: os.Getenv("BOOTSTRAP_ADMIN_DISPLAY_NAME"),
		AdminPassword:    os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
	})
}

func NormalizeConfig(cfg Config) (Config, error) {
	cfg.WorkspaceName = strings.Join(strings.Fields(strings.TrimSpace(cfg.WorkspaceName)), " ")
	cfg.AdminEmail = strings.ToLower(strings.TrimSpace(cfg.AdminEmail))
	cfg.AdminUsername = strings.ToLower(strings.TrimSpace(cfg.AdminUsername))
	cfg.AdminDisplayName = strings.Join(strings.Fields(strings.TrimSpace(cfg.AdminDisplayName)), " ")
	cfg.AdminPassword = strings.TrimSpace(cfg.AdminPassword)

	var problems []string
	if cfg.WorkspaceName == "" {
		problems = append(problems, "BOOTSTRAP_WORKSPACE_NAME is required")
	} else if len([]rune(cfg.WorkspaceName)) > 80 {
		problems = append(problems, "BOOTSTRAP_WORKSPACE_NAME must be 80 characters or fewer")
	}
	if !emailPattern.MatchString(cfg.AdminEmail) {
		problems = append(problems, "BOOTSTRAP_ADMIN_EMAIL is invalid")
	}
	if !usernamePattern.MatchString(cfg.AdminUsername) {
		problems = append(problems, "BOOTSTRAP_ADMIN_USERNAME must be 3-32 characters and contain lowercase letters, numbers, underscores, or hyphens")
	}
	if cfg.AdminDisplayName == "" {
		problems = append(problems, "BOOTSTRAP_ADMIN_DISPLAY_NAME is required")
	} else if len([]rune(cfg.AdminDisplayName)) > 80 {
		problems = append(problems, "BOOTSTRAP_ADMIN_DISPLAY_NAME must be 80 characters or fewer")
	}
	if len(cfg.AdminPassword) < 8 {
		problems = append(problems, "BOOTSTRAP_ADMIN_PASSWORD must be at least 8 characters")
	} else if len(cfg.AdminPassword) > 128 {
		problems = append(problems, "BOOTSTRAP_ADMIN_PASSWORD must be 128 characters or fewer")
	}

	if len(problems) > 0 {
		return Config{}, errors.New(strings.Join(problems, "; "))
	}
	return cfg, nil
}

func (s *Service) Bootstrap(ctx context.Context, cfg Config) (Result, error) {
	cfg, err := NormalizeConfig(cfg)
	if err != nil {
		return Result{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return Result{}, fmt.Errorf("hash admin password: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("begin bootstrap transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Exclusive locks make the empty-database check and first-admin creation atomic.
	if _, err := tx.Exec(ctx, `LOCK TABLE workspaces, users, workspace_members IN ACCESS EXCLUSIVE MODE`); err != nil {
		return Result{}, fmt.Errorf("lock bootstrap tables: %w", err)
	}

	var hasExistingData bool
	if err := tx.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM workspaces)
			OR EXISTS (SELECT 1 FROM users)
			OR EXISTS (SELECT 1 FROM workspace_members)
	`).Scan(&hasExistingData); err != nil {
		return Result{}, fmt.Errorf("check bootstrap state: %w", err)
	}
	if hasExistingData {
		return Result{}, ErrDatabaseNotEmpty
	}

	var result Result
	if err := tx.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ($1)
		RETURNING id::text
	`, cfg.WorkspaceName).Scan(&result.WorkspaceID); err != nil {
		return Result{}, fmt.Errorf("create workspace: %w", err)
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name, is_active)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id::text
	`, cfg.AdminEmail, cfg.AdminUsername, string(passwordHash), cfg.AdminDisplayName).Scan(&result.AdminUserID); err != nil {
		return Result{}, fmt.Errorf("create admin user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin')
	`, result.WorkspaceID, result.AdminUserID); err != nil {
		return Result{}, fmt.Errorf("create admin membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit bootstrap transaction: %w", err)
	}
	return result, nil
}
