package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// SavedQuery is the structured form of a predefined query: a .toml file
// alongside plain .sql files in the query directory. Plain .sql is enough
// for a one-liner; .toml lets a query carry a name/description with it.
type SavedQuery struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	SQL         string `toml:"sql"`
}

// queryDir resolves where predefined queries live, in priority order:
// --query-dir flag > SNOWSTORM_QUERY_DIR env var > ~/.snowstorm/queries.
func queryDir() string {
	if flagQueryDir != "" {
		return flagQueryDir
	}
	if v := os.Getenv("SNOWSTORM_QUERY_DIR"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".snowstorm", "queries")
}

// resolveSavedQuery looks up a predefined query by name: <dir>/<name>.sql
// (plain text) or <dir>/<name>.toml (structured, with a [sql] field) -- the
// .sql file wins if both exist.
func resolveSavedQuery(name string) (string, error) {
	dir := queryDir()

	sqlPath := filepath.Join(dir, name+".sql")
	if b, err := os.ReadFile(sqlPath); err == nil {
		return string(b), nil
	}

	tomlPath := filepath.Join(dir, name+".toml")
	b, err := os.ReadFile(tomlPath)
	if err != nil {
		return "", fmt.Errorf("saved query %q not found in %s (looked for %s.sql and %s.toml)", name, dir, name, name)
	}

	var sq SavedQuery
	if _, err := toml.Decode(string(b), &sq); err != nil {
		return "", fmt.Errorf("parse saved query %q: %w", tomlPath, err)
	}
	if strings.TrimSpace(sq.SQL) == "" {
		return "", fmt.Errorf("saved query %q: %s has no sql field", name, tomlPath)
	}
	return sq.SQL, nil
}

// savedQueryInfo describes one entry for `snowstorm queries list`.
type savedQueryInfo struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"` // "sql" or "toml"
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

// listSavedQueries scans queryDir() for .sql and .toml query definitions.
func listSavedQueries() ([]savedQueryInfo, error) {
	dir := queryDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []savedQueryInfo{}, nil
		}
		return nil, fmt.Errorf("read query dir %q: %w", dir, err)
	}

	var out []savedQueryInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		path := filepath.Join(dir, name)

		switch ext {
		case ".sql":
			desc := ""
			if b, err := os.ReadFile(path); err == nil {
				desc = firstCommentLine(string(b))
			}
			out = append(out, savedQueryInfo{Name: base, Kind: "sql", Path: path, Description: desc})
		case ".toml":
			var sq SavedQuery
			desc := ""
			if b, err := os.ReadFile(path); err == nil {
				if _, err := toml.Decode(string(b), &sq); err == nil {
					desc = sq.Description
				}
			}
			out = append(out, savedQueryInfo{Name: base, Kind: "toml", Path: path, Description: desc})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// firstCommentLine returns the first "-- ..." line of a .sql file, trimmed,
// used as a lightweight description when listing plain-text saved queries.
func firstCommentLine(sqlText string) string {
	for line := range strings.SplitSeq(sqlText, "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "--"); ok {
			return strings.TrimSpace(rest)
		}
		if line != "" {
			break
		}
	}
	return ""
}
