package postgresenv

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

var variableNames = []string{
	"PGHOST",
	"PGPORT",
	"PGDATABASE",
	"PGUSER",
	"PGPASSWORD",
	"PGSSLMODE",
	"PGCONNECT_TIMEOUT",
}

func FromURL(databaseURL string) ([]string, error) {
	parsed, err := url.Parse(strings.TrimSpace(databaseURL))
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return nil, errors.New("database URL must be a valid PostgreSQL URL")
	}
	database, err := url.PathUnescape(strings.TrimPrefix(parsed.EscapedPath(), "/"))
	if err != nil || parsed.Hostname() == "" || database == "" {
		return nil, errors.New("database URL must include a host and database name")
	}
	port := parsed.Port()
	if port == "" {
		port = "5432"
	}
	user := ""
	password := ""
	if parsed.User != nil {
		user = parsed.User.Username()
		password, _ = parsed.User.Password()
	}
	sslMode := parsed.Query().Get("sslmode")
	if sslMode == "" {
		sslMode = "disable"
	}
	return append(Without(os.Environ(), variableNames...),
		"PGHOST="+parsed.Hostname(),
		"PGPORT="+port,
		"PGDATABASE="+database,
		"PGUSER="+user,
		"PGPASSWORD="+password,
		"PGSSLMODE="+sslMode,
		"PGCONNECT_TIMEOUT=10",
	), nil
}

func Without(values []string, names ...string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		keep := true
		for _, name := range names {
			if strings.HasPrefix(value, name+"=") {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, value)
		}
	}
	return filtered
}
