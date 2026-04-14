package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FileConfig holds values loaded from an optional user config file.
// Absent fields are represented as empty / nil so callers can distinguish
// "unset" from "set to zero value" and apply precedence correctly.
type FileConfig struct {
	Theme             string
	Repo              string
	File              string
	NoCommit          *bool
	HideDoneAfterDays *int
}

// DefaultPath returns the platform-appropriate config file location:
//   - macOS:   ~/Library/Application Support/doit/config.toml
//   - Linux:   $XDG_CONFIG_HOME/doit/config.toml (or ~/.config/...)
//   - Windows: %AppData%\doit\config.toml
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "doit", "config.toml"), nil
}

// Load reads the config file at path. A missing file is not an error.
// Format: simple `key = value` lines, `#` or `;` comments, optional quotes.
func Load(path string) (FileConfig, error) {
	var fc FileConfig
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fc, nil
		}
		return fc, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	lineno := 0
	for sc.Scan() {
		lineno++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return fc, fmt.Errorf("%s:%d: expected key = value", path, lineno)
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if len(val) >= 2 && (val[0] == '"' && val[len(val)-1] == '"') {
			val = val[1 : len(val)-1]
		}
		switch strings.ToLower(key) {
		case "theme":
			fc.Theme = val
		case "repo":
			fc.Repo = expandHome(val)
		case "file":
			fc.File = val
		case "no_commit", "nocommit":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return fc, fmt.Errorf("%s:%d: no_commit must be true/false", path, lineno)
			}
			fc.NoCommit = &b
		case "hide_done_after_days":
			n, err := strconv.Atoi(val)
			if err != nil || n < 0 {
				return fc, fmt.Errorf("%s:%d: hide_done_after_days must be a non-negative integer", path, lineno)
			}
			fc.HideDoneAfterDays = &n
		default:
			return fc, fmt.Errorf("%s:%d: unknown key %q", path, lineno, key)
		}
	}
	return fc, sc.Err()
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
