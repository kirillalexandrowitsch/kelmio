package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const MaxLimit = 100

var ErrInvalid = errors.New("invalid pagination")

type Params struct {
	Limit  int
	Offset int
}

type cursorPayload struct {
	Offset int `json:"offset"`
}

func Default(defaultLimit int) Params {
	if defaultLimit < 1 {
		defaultLimit = MaxLimit
	}
	if defaultLimit > MaxLimit {
		defaultLimit = MaxLimit
	}

	return Params{Limit: defaultLimit}
}

func Parse(query url.Values, defaultLimit int) (Params, error) {
	params := Default(defaultLimit)

	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit < 1 || limit > MaxLimit {
			return Params{}, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalid, MaxLimit)
		}
		params.Limit = limit
	}

	if rawCursor := strings.TrimSpace(query.Get("cursor")); rawCursor != "" {
		offset, err := DecodeCursor(rawCursor)
		if err != nil {
			return Params{}, err
		}
		params.Offset = offset
	}

	return params, nil
}

func DecodeCursor(rawCursor string) (int, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(rawCursor)
	if err != nil {
		return 0, fmt.Errorf("%w: cursor is malformed", ErrInvalid)
	}

	var payload cursorPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return 0, fmt.Errorf("%w: cursor is malformed", ErrInvalid)
	}
	if payload.Offset < 0 {
		return 0, fmt.Errorf("%w: cursor offset must be non-negative", ErrInvalid)
	}

	return payload.Offset, nil
}

func EncodeCursor(offset int) (string, error) {
	if offset < 0 {
		return "", fmt.Errorf("%w: cursor offset must be non-negative", ErrInvalid)
	}

	encoded, err := json.Marshal(cursorPayload{Offset: offset})
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func Window[T any](items []T, params Params) ([]T, *string, error) {
	if params.Limit < 1 {
		params.Limit = MaxLimit
	}

	if len(items) <= params.Limit {
		return items, nil, nil
	}

	nextCursor, err := EncodeCursor(params.Offset + params.Limit)
	if err != nil {
		return nil, nil, err
	}

	return items[:params.Limit], &nextCursor, nil
}
