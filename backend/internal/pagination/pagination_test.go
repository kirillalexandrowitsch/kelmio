package pagination

import (
	"net/url"
	"testing"
)

func TestParseDefaults(t *testing.T) {
	t.Parallel()

	got, err := Parse(url.Values{}, 50)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Limit != 50 {
		t.Fatalf("Limit = %d, want 50", got.Limit)
	}
	if got.Offset != 0 {
		t.Fatalf("Offset = %d, want 0", got.Offset)
	}
}

func TestParseExplicitLimitAndCursor(t *testing.T) {
	t.Parallel()

	cursor, err := EncodeCursor(25)
	if err != nil {
		t.Fatalf("EncodeCursor(): %v", err)
	}

	got, err := Parse(url.Values{
		"limit":  {"10"},
		"cursor": {cursor},
	}, 50)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Limit != 10 {
		t.Fatalf("Limit = %d, want 10", got.Limit)
	}
	if got.Offset != 25 {
		t.Fatalf("Offset = %d, want 25", got.Offset)
	}
}

func TestParseRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	tests := []string{"0", "101", "abc"}
	for _, rawLimit := range tests {
		rawLimit := rawLimit
		t.Run(rawLimit, func(t *testing.T) {
			t.Parallel()

			if _, err := Parse(url.Values{"limit": {rawLimit}}, 50); err == nil {
				t.Fatal("Parse() error = nil, want invalid limit error")
			}
		})
	}
}

func TestParseRejectsInvalidCursor(t *testing.T) {
	t.Parallel()

	tests := []string{"not-base64url", "eyJvZmZzZXQiOi0xfQ"}
	for _, rawCursor := range tests {
		rawCursor := rawCursor
		t.Run(rawCursor, func(t *testing.T) {
			t.Parallel()

			if _, err := Parse(url.Values{"cursor": {rawCursor}}, 50); err == nil {
				t.Fatal("Parse() error = nil, want invalid cursor error")
			}
		})
	}
}

func TestWindowReturnsNextCursor(t *testing.T) {
	t.Parallel()

	items, nextCursor, err := Window([]int{1, 2, 3}, Params{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("Window() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if nextCursor == nil {
		t.Fatal("nextCursor is nil, want cursor")
	}

	offset, err := DecodeCursor(*nextCursor)
	if err != nil {
		t.Fatalf("DecodeCursor(): %v", err)
	}
	if offset != 6 {
		t.Fatalf("next offset = %d, want 6", offset)
	}
}

func TestWindowReturnsNilCursorOnLastPage(t *testing.T) {
	t.Parallel()

	items, nextCursor, err := Window([]int{1, 2}, Params{Limit: 2})
	if err != nil {
		t.Fatalf("Window() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if nextCursor != nil {
		t.Fatalf("nextCursor = %q, want nil", *nextCursor)
	}
}
