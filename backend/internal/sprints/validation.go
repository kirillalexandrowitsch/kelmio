package sprints

import (
	"errors"
	"strings"
	"time"
)

func normalizeCreateSprint(req createSprintRequest) (normalizedCreateSprint, error) {
	projectID, err := normalizeID(req.ProjectID, "project_id")
	if err != nil {
		return normalizedCreateSprint{}, err
	}

	input := normalizedCreateSprint{
		ProjectID: projectID,
		Name:      strings.TrimSpace(req.Name),
		Goal:      strings.TrimSpace(req.Goal),
		StartDate: strings.TrimSpace(req.StartDate),
		EndDate:   strings.TrimSpace(req.EndDate),
	}

	if err := validateSprintDetails(input.Name, input.Goal, input.StartDate, input.EndDate); err != nil {
		return input, err
	}

	return input, nil
}

func normalizeUpdateSprint(req updateSprintRequest) (normalizedUpdateSprint, error) {
	input := normalizedUpdateSprint{
		Name:      strings.TrimSpace(req.Name),
		Goal:      strings.TrimSpace(req.Goal),
		StartDate: strings.TrimSpace(req.StartDate),
		EndDate:   strings.TrimSpace(req.EndDate),
	}

	if err := validateSprintDetails(input.Name, input.Goal, input.StartDate, input.EndDate); err != nil {
		return input, err
	}

	return input, nil
}

func validateSprintDetails(name string, goal string, startDate string, endDate string) error {
	if name == "" {
		return errors.New("name is required")
	}
	if len(name) > 120 {
		return errors.New("name must be 120 characters or fewer")
	}
	if len(goal) > 1000 {
		return errors.New("goal must be 1000 characters or fewer")
	}

	var parsedStartDate time.Time
	var parsedEndDate time.Time
	if startDate != "" {
		date, err := time.Parse(time.DateOnly, startDate)
		if err != nil {
			return errors.New("start_date must be YYYY-MM-DD")
		}
		parsedStartDate = date
	}
	if endDate != "" {
		date, err := time.Parse(time.DateOnly, endDate)
		if err != nil {
			return errors.New("end_date must be YYYY-MM-DD")
		}
		parsedEndDate = date
	}
	if !parsedStartDate.IsZero() && !parsedEndDate.IsZero() && parsedStartDate.After(parsedEndDate) {
		return errors.New("start_date must be before or equal to end_date")
	}

	return nil
}

func normalizeID(id string, field string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New(field + " is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New(field + " is invalid")
	}

	return id, nil
}

func normalizeOptionalID(id string, field string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", nil
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New(field + " is invalid")
	}

	return id, nil
}
