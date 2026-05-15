package migrations

import (
	"strings"
	"testing"
)

func TestExtractUpSQLWithGooseSections(t *testing.T) {
	content := `-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id UUID PRIMARY KEY
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE users;`

	got := extractUpSQL(content)

	if !strings.Contains(got, "CREATE TABLE users") {
		t.Fatalf("expected up sql to contain create statement, got %q", got)
	}
	if strings.Contains(got, "DROP TABLE users") {
		t.Fatalf("expected up sql to exclude down statement, got %q", got)
	}
}

func TestExtractUpSQLWithoutGooseSections(t *testing.T) {
	content := "CREATE TABLE users (id UUID PRIMARY KEY);"

	got := extractUpSQL(content)
	if got != content {
		t.Fatalf("expected content to be unchanged, got %q", got)
	}
}
