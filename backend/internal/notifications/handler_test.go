package notifications

import (
	"reflect"
	"testing"
)

func TestMentionUsernames(t *testing.T) {
	t.Parallel()

	got := mentionUsernames("Ping @Demo_Member and @demo_member, also @QA-Lead.")
	want := []string{"demo_member", "qa-lead"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mentionUsernames = %#v, want %#v", got, want)
	}
}

func TestCommentNotificationRecipients(t *testing.T) {
	t.Parallel()

	got := commentNotificationRecipients("actor", "reporter", "assignee", []string{
		"mentioned",
		"assignee",
		"actor",
	})
	want := []commentRecipient{
		{UserID: "mentioned", NotificationType: TypeIssueMentioned},
		{UserID: "assignee", NotificationType: TypeIssueMentioned},
		{UserID: "reporter", NotificationType: TypeIssueCommented},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("recipients = %#v, want %#v", got, want)
	}
}

func TestCommentNotificationRecipientsNoSelfNotification(t *testing.T) {
	t.Parallel()

	got := commentNotificationRecipients("actor", "actor", "actor", []string{"actor"})
	if len(got) != 0 {
		t.Fatalf("recipients = %#v, want none", got)
	}
}

func TestCommentPreview(t *testing.T) {
	t.Parallel()

	got := commentPreview("  hello   world  ")
	if got != "hello world" {
		t.Fatalf("preview = %q, want hello world", got)
	}

	long := commentPreview("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz")
	if len([]rune(long)) != 83 {
		t.Fatalf("long preview length = %d, want 83", len([]rune(long)))
	}
}

func TestNormalizeNotificationID(t *testing.T) {
	t.Parallel()

	got, err := normalizeNotificationID(" 6D5257D4-002E-44DA-8925-D9108699C504 ")
	if err != nil {
		t.Fatalf("normalize id: %v", err)
	}
	if got != "6d5257d4-002e-44da-8925-d9108699c504" {
		t.Fatalf("id = %q", got)
	}

	if _, err := normalizeNotificationID("not-a-uuid"); err == nil {
		t.Fatal("expected invalid id error")
	}
}
