package projectmembers

import (
	"errors"
	"regexp"
	"strings"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var validProjectRoles = map[string]bool{
	"lead":        true,
	"contributor": true,
	"viewer":      true,
}

func normalizeID(value string, field string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", errors.New(field + " is required")
	}
	if !uuidPattern.MatchString(value) {
		return "", errors.New(field + " must be a valid id")
	}
	return value, nil
}

func normalizeRole(role string) (string, error) {
	role = strings.TrimSpace(role)
	if !validProjectRoles[role] {
		return "", errors.New("role is invalid")
	}
	return role, nil
}
