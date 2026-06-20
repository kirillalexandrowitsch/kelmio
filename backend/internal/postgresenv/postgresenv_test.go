package postgresenv

import (
	"strings"
	"testing"
)

func TestFromURLSeparatesConnectionFields(t *testing.T) {
	t.Setenv("PGPASSWORD", "old-password")
	environment, err := FromURL("postgres://backup-user:p%40ss%3Aword@database.internal:15432/task_tracker?sslmode=require")
	if err != nil {
		t.Fatalf("FromURL() error = %v", err)
	}
	values := make(map[string]string, len(environment))
	for _, item := range environment {
		key, value, found := strings.Cut(item, "=")
		if found {
			values[key] = value
		}
	}
	if values["PGHOST"] != "database.internal" || values["PGPORT"] != "15432" {
		t.Fatalf("unexpected host environment: %#v", values)
	}
	if values["PGDATABASE"] != "task_tracker" || values["PGUSER"] != "backup-user" {
		t.Fatalf("unexpected database environment: %#v", values)
	}
	if values["PGPASSWORD"] != "p@ss:word" || values["PGSSLMODE"] != "require" {
		t.Fatalf("unexpected secure connection environment: %#v", values)
	}
}

func TestFromURLRejectsInvalidURL(t *testing.T) {
	for _, databaseURL := range []string{"", "mysql://localhost/tasks", "postgres:///tasks", "postgres://localhost"} {
		if _, err := FromURL(databaseURL); err == nil {
			t.Fatalf("expected %q to be rejected", databaseURL)
		}
	}
}
