package workflows

import (
	"errors"
	"regexp"
	"strings"
)

var workflowKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
var workflowColorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)
var workflowUUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var validWorkflowCategories = map[string]bool{
	"backlog":     true,
	"todo":        true,
	"in_progress": true,
	"done":        true,
}

func normalizeCreateStatus(req createStatusRequest) (normalizedCreateStatus, error) {
	input := normalizedCreateStatus{
		Key:      strings.ToLower(strings.TrimSpace(req.Key)),
		Name:     strings.TrimSpace(req.Name),
		Color:    strings.ToLower(strings.TrimSpace(req.Color)),
		Category: strings.TrimSpace(req.Category),
	}
	if err := validateStatusDetails(input.Key, input.Name, input.Color, input.Category); err != nil {
		return input, err
	}
	return input, nil
}

func normalizeUpdateStatus(req updateStatusRequest) (normalizedUpdateStatus, error) {
	input := normalizedUpdateStatus{}
	if req.Name != nil {
		input.HasName = true
		input.Name = strings.TrimSpace(*req.Name)
		if input.Name == "" {
			return input, errors.New("name is required")
		}
		if len(input.Name) > 60 {
			return input, errors.New("name must be 60 characters or fewer")
		}
	}
	if req.Color != nil {
		input.HasColor = true
		input.Color = strings.ToLower(strings.TrimSpace(*req.Color))
		if !workflowColorPattern.MatchString(input.Color) {
			return input, errors.New("color must be a valid #RRGGBB value")
		}
	}
	if req.Category != nil {
		input.HasCategory = true
		input.Category = strings.TrimSpace(*req.Category)
		if !validWorkflowCategories[input.Category] {
			return input, errors.New("category is invalid")
		}
	}
	if !input.HasName && !input.HasColor && !input.HasCategory {
		return input, errors.New("at least one status field is required")
	}
	return input, nil
}

func normalizeStatusOrder(req statusOrderRequest) ([]string, error) {
	if len(req.StatusIDs) == 0 {
		return nil, errors.New("status_ids is required")
	}
	return normalizeUniqueIDs(req.StatusIDs, "status_ids")
}

func normalizeTransitions(req replaceTransitionsRequest) ([]normalizedTransition, error) {
	transitions := make([]normalizedTransition, 0, len(req.Transitions))
	seen := make(map[string]bool, len(req.Transitions))
	for _, transition := range req.Transitions {
		fromID, err := normalizeWorkflowID(transition.FromStatusID, "from_status_id")
		if err != nil {
			return nil, err
		}
		toID, err := normalizeWorkflowID(transition.ToStatusID, "to_status_id")
		if err != nil {
			return nil, err
		}
		if fromID == toID {
			return nil, errors.New("workflow transition cannot target the same status")
		}
		key := fromID + ":" + toID
		if seen[key] {
			return nil, errors.New("workflow transitions must not contain duplicates")
		}
		seen[key] = true
		transitions = append(transitions, normalizedTransition{
			FromStatusID: fromID,
			ToStatusID:   toID,
		})
	}
	return transitions, nil
}

func normalizeWorkflowID(value string, field string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", errors.New(field + " is required")
	}
	if !workflowUUIDPattern.MatchString(value) {
		return "", errors.New(field + " must be a valid id")
	}
	return value, nil
}

func normalizeUniqueIDs(values []string, field string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		id, err := normalizeWorkflowID(value, field)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			return nil, errors.New(field + " must not contain duplicates")
		}
		seen[id] = true
		normalized = append(normalized, id)
	}
	return normalized, nil
}

func validateStatusDetails(key string, name string, color string, category string) error {
	if !workflowKeyPattern.MatchString(key) {
		return errors.New("key must be a lowercase identifier with 1-32 letters, numbers, or underscores")
	}
	if name == "" {
		return errors.New("name is required")
	}
	if len(name) > 60 {
		return errors.New("name must be 60 characters or fewer")
	}
	if !workflowColorPattern.MatchString(color) {
		return errors.New("color must be a valid #RRGGBB value")
	}
	if !validWorkflowCategories[category] {
		return errors.New("category is invalid")
	}
	return nil
}
