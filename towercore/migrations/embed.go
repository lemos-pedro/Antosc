package migrations

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed *.sql
var files embed.FS

type Script struct {
	Name string
	SQL  string
}

func Scripts() ([]Script, error) {
	entries, err := files.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	scripts := make([]Script, 0, len(names))
	for _, name := range names {
		content, err := files.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}

		sql := extractUpSQL(string(content))
		if strings.TrimSpace(sql) == "" {
			continue
		}
		scripts = append(scripts, Script{
			Name: name,
			SQL:  sql,
		})
	}

	return scripts, nil
}

func extractUpSQL(content string) string {
	lines := strings.Split(content, "\n")
	hasGooseUp := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "-- +goose Up") {
			hasGooseUp = true
			break
		}
	}
	if !hasGooseUp {
		return content
	}

	var kept []string
	inUp := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "-- +goose Up"):
			inUp = true
			kept = append(kept, line)
		case strings.HasPrefix(trimmed, "-- +goose Down"):
			inUp = false
			return strings.TrimSpace(strings.Join(kept, "\n"))
		case inUp:
			kept = append(kept, line)
		}
	}

	return strings.TrimSpace(strings.Join(kept, "\n"))
}
