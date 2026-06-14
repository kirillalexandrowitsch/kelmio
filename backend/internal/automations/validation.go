package automations

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var validTriggers = map[string]bool{
	"issue_created":    true,
	"status_changed":   true,
	"assignee_changed": true,
	"priority_changed": true,
}

var validIssueTypes = map[string]bool{
	"task": true, "bug": true, "story": true, "epic": true, "subtask": true,
}

var validPriorities = map[string]bool{
	"low": true, "medium": true, "high": true, "critical": true,
}

type normalizedDefinition struct {
	Conditions   []map[string]any
	Actions      []map[string]any
	Dependencies []dependency
}

type dependency struct {
	Kind string
	ID   string
}

func normalizeCreateRule(req createRuleRequest) (normalizedCreateRule, error) {
	name, err := normalizeName(req.Name)
	if err != nil {
		return normalizedCreateRule{}, err
	}
	trigger, err := normalizeTrigger(req.TriggerType)
	if err != nil {
		return normalizedCreateRule{}, err
	}
	definition, conditions, actions, err := normalizeDefinition(req.Conditions, req.Actions, true, true)
	if err != nil {
		return normalizedCreateRule{}, err
	}
	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}
	return normalizedCreateRule{
		Name: name, TriggerType: trigger, Conditions: conditions, Actions: actions,
		IsEnabled: enabled, Definition: definition,
	}, nil
}

func normalizeUpdateRule(req updateRuleRequest) (normalizedUpdateRule, error) {
	input := normalizedUpdateRule{}
	if req.Name != nil {
		input.HasName = true
		name, err := normalizeName(*req.Name)
		if err != nil {
			return input, err
		}
		input.Name = name
	}
	if req.TriggerType != nil {
		input.HasTriggerType = true
		trigger, err := normalizeTrigger(*req.TriggerType)
		if err != nil {
			return input, err
		}
		input.TriggerType = trigger
	}
	input.HasConditions = req.Conditions != nil
	input.HasActions = req.Actions != nil
	if input.HasConditions || input.HasActions {
		definition, conditions, actions, err := normalizeDefinition(req.Conditions, req.Actions, input.HasConditions, input.HasActions)
		if err != nil {
			return input, err
		}
		input.Definition = definition
		input.Conditions = conditions
		input.Actions = actions
	}
	if req.IsEnabled != nil {
		input.HasIsEnabled = true
		input.IsEnabled = *req.IsEnabled
	}
	if !input.HasName && !input.HasTriggerType && !input.HasConditions && !input.HasActions && !input.HasIsEnabled {
		return input, errors.New("at least one automation rule field is required")
	}
	return input, nil
}

func normalizeDefinition(
	conditionsRaw json.RawMessage,
	actionsRaw json.RawMessage,
	hasConditions bool,
	hasActions bool,
) (normalizedDefinition, json.RawMessage, json.RawMessage, error) {
	definition := normalizedDefinition{}
	var conditionsJSON json.RawMessage
	var actionsJSON json.RawMessage
	if hasConditions {
		items, dependencies, err := normalizeItems(conditionsRaw, true)
		if err != nil {
			return definition, nil, nil, fmt.Errorf("conditions: %w", err)
		}
		definition.Conditions = items
		definition.Dependencies = append(definition.Dependencies, dependencies...)
		conditionsJSON, _ = json.Marshal(items)
	}
	if hasActions {
		items, dependencies, err := normalizeItems(actionsRaw, false)
		if err != nil {
			return definition, nil, nil, fmt.Errorf("actions: %w", err)
		}
		if len(items) == 0 {
			return definition, nil, nil, errors.New("actions must contain at least one action")
		}
		definition.Actions = items
		definition.Dependencies = append(definition.Dependencies, dependencies...)
		actionsJSON, _ = json.Marshal(items)
	}
	return definition, conditionsJSON, actionsJSON, nil
}

func normalizeItems(raw json.RawMessage, conditions bool) ([]map[string]any, []dependency, error) {
	if raw == nil {
		if conditions {
			raw = json.RawMessage(`[]`)
		} else {
			return nil, nil, errors.New("is required")
		}
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, nil, errors.New("must be an array of objects")
	}
	var items []map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	if err := decoder.Decode(&items); err != nil {
		return nil, nil, errors.New("must be an array of objects")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, nil, errors.New("must contain a single array")
	}
	if len(items) > 20 {
		return nil, nil, errors.New("must contain 20 items or fewer")
	}
	normalized := make([]map[string]any, 0, len(items))
	dependencies := make([]dependency, 0)
	seenConditions := map[string]bool{}
	seenLabels := map[string]bool{}
	for _, item := range items {
		itemType, err := requiredString(item, "type")
		if err != nil {
			return nil, nil, err
		}
		itemType = strings.TrimSpace(itemType)
		var normalizedItem map[string]any
		var itemDependencies []dependency
		if conditions {
			normalizedItem, itemDependencies, err = normalizeCondition(itemType, item)
			if err == nil && itemType != "label" {
				if seenConditions[itemType] {
					err = errors.New("scalar condition types must not contain duplicates")
				}
				seenConditions[itemType] = true
			}
			if err == nil && itemType == "label" {
				labelID := normalizedItem["label_id"].(string)
				if seenLabels[labelID] {
					err = errors.New("label conditions must reference different labels")
				}
				seenLabels[labelID] = true
			}
		} else {
			normalizedItem, itemDependencies, err = normalizeAction(itemType, item)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", itemType, err)
		}
		normalized = append(normalized, normalizedItem)
		dependencies = append(dependencies, itemDependencies...)
	}
	return normalized, dependencies, nil
}

func normalizeCondition(itemType string, item map[string]json.RawMessage) (map[string]any, []dependency, error) {
	switch itemType {
	case "issue_type":
		value, err := enumValue(item, validIssueTypes)
		return typedValue(itemType, value), nil, requireExactKeys(item, err, "type", "value")
	case "priority":
		value, err := enumValue(item, validPriorities)
		return typedValue(itemType, value), nil, requireExactKeys(item, err, "type", "value")
	case "workflow_status":
		id, err := requiredID(item, "workflow_status_id")
		return typedID(itemType, "workflow_status_id", id), []dependency{{Kind: "workflow_status", ID: id}}, requireExactKeys(item, err, "type", "workflow_status_id")
	case "assignee":
		id, err := nullableID(item, "user_id")
		deps := optionalDependency("user", id)
		return typedNullableID(itemType, "user_id", id), deps, requireExactKeys(item, err, "type", "user_id")
	case "reporter":
		id, err := requiredID(item, "user_id")
		return typedID(itemType, "user_id", id), []dependency{{Kind: "user", ID: id}}, requireExactKeys(item, err, "type", "user_id")
	case "label":
		id, err := requiredID(item, "label_id")
		return typedID(itemType, "label_id", id), []dependency{{Kind: "label", ID: id}}, requireExactKeys(item, err, "type", "label_id")
	default:
		return nil, nil, errors.New("type is invalid")
	}
}

func normalizeAction(itemType string, item map[string]json.RawMessage) (map[string]any, []dependency, error) {
	switch itemType {
	case "change_workflow_status":
		id, err := requiredID(item, "workflow_status_id")
		return typedID(itemType, "workflow_status_id", id), []dependency{{Kind: "workflow_status", ID: id}}, requireExactKeys(item, err, "type", "workflow_status_id")
	case "change_assignee":
		id, err := nullableID(item, "user_id")
		return typedNullableID(itemType, "user_id", id), optionalDependency("user", id), requireExactKeys(item, err, "type", "user_id")
	case "change_priority":
		value, err := enumValue(item, validPriorities)
		return typedValue(itemType, value), nil, requireExactKeys(item, err, "type", "value")
	case "add_label", "remove_label":
		id, err := requiredID(item, "label_id")
		return typedID(itemType, "label_id", id), []dependency{{Kind: "label", ID: id}}, requireExactKeys(item, err, "type", "label_id")
	default:
		return nil, nil, errors.New("type is invalid")
	}
}

func normalizeRuleOrder(req ruleOrderRequest) ([]string, error) {
	if len(req.RuleIDs) == 0 {
		return nil, errors.New("rule_ids is required")
	}
	ids := make([]string, 0, len(req.RuleIDs))
	seen := map[string]bool{}
	for _, value := range req.RuleIDs {
		id, err := normalizeID(value, "rule_ids")
		if err != nil {
			return nil, err
		}
		if seen[id] {
			return nil, errors.New("rule_ids must not contain duplicates")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}

func normalizeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("name is required")
	}
	if len([]rune(value)) > 100 {
		return "", errors.New("name must be 100 characters or fewer")
	}
	return value, nil
}

func normalizeTrigger(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !validTriggers[value] {
		return "", errors.New("trigger_type is invalid")
	}
	return value, nil
}

func normalizeID(value string, field string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if !uuidPattern.MatchString(value) {
		return "", errors.New(field + " must be a valid id")
	}
	return value, nil
}

func requiredString(item map[string]json.RawMessage, key string) (string, error) {
	raw, ok := item[key]
	if !ok {
		return "", errors.New(key + " is required")
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
		return "", errors.New(key + " must be a non-empty string")
	}
	return value, nil
}

func requiredID(item map[string]json.RawMessage, key string) (string, error) {
	value, err := requiredString(item, key)
	if err != nil {
		return "", err
	}
	return normalizeID(value, key)
}

func nullableID(item map[string]json.RawMessage, key string) (*string, error) {
	raw, ok := item[key]
	if !ok {
		return nil, errors.New(key + " is required")
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, errors.New(key + " must be a valid id or null")
	}
	id, err := normalizeID(value, key)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func enumValue(item map[string]json.RawMessage, allowed map[string]bool) (string, error) {
	value, err := requiredString(item, "value")
	value = strings.TrimSpace(value)
	if err == nil && !allowed[value] {
		err = errors.New("value is invalid")
	}
	return value, err
}

func requireExactKeys(item map[string]json.RawMessage, prior error, keys ...string) error {
	if prior != nil {
		return prior
	}
	if len(item) != len(keys) {
		return errors.New("contains unsupported fields")
	}
	for _, key := range keys {
		if _, ok := item[key]; !ok {
			return errors.New(key + " is required")
		}
	}
	return nil
}

func typedValue(itemType string, value string) map[string]any {
	return map[string]any{"type": itemType, "value": value}
}

func typedID(itemType string, key string, id string) map[string]any {
	return map[string]any{"type": itemType, key: id}
}

func typedNullableID(itemType string, key string, id *string) map[string]any {
	return map[string]any{"type": itemType, key: id}
}

func optionalDependency(kind string, id *string) []dependency {
	if id == nil {
		return nil
	}
	return []dependency{{Kind: kind, ID: *id}}
}
