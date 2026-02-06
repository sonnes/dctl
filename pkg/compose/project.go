package compose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectState represents the persisted state of a compose project.
type ProjectState struct {
	Name        string            `json:"name"`
	ComposeFile string            `json:"compose_file"`
	ProjectDir  string            `json:"project_dir"`
	Containers  map[string]string `json:"containers"`  // service name → container ID
	Networks    []string          `json:"networks"`     // created network names
	Volumes     []string          `json:"volumes"`      // created volume names
}

// projectsDir returns the path to the projects state directory.
func projectsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".dctl", "projects"), nil
}

// projectFilePath returns the path to a project's state file.
func projectFilePath(name string) (string, error) {
	dir, err := projectsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".json"), nil
}

// SaveProject writes project state to disk.
func SaveProject(state *ProjectState) error {
	dir, err := projectsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating projects directory: %w", err)
	}

	path, err := projectFilePath(state.Name)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling project state: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing project state: %w", err)
	}
	return nil
}

// LoadProject reads project state from disk.
func LoadProject(name string) (*ProjectState, error) {
	path, err := projectFilePath(name)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project %q not found", name)
		}
		return nil, fmt.Errorf("reading project state: %w", err)
	}

	var state ProjectState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing project state: %w", err)
	}
	return &state, nil
}

// DeleteProject removes project state from disk.
func DeleteProject(name string) error {
	path, err := projectFilePath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing project state: %w", err)
	}
	return nil
}

// ListProjects returns the names of all saved projects.
func ListProjects() ([]string, error) {
	dir, err := projectsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading projects directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
	}
	return names, nil
}

// ResolveProjectName determines the project name from flag, compose file, or directory name.
func ResolveProjectName(flagName string, composeFile *ComposeFile, projectDir string) string {
	if flagName != "" {
		return sanitizeProjectName(flagName)
	}
	if composeFile != nil && composeFile.Name != "" {
		return sanitizeProjectName(composeFile.Name)
	}
	return sanitizeProjectName(filepath.Base(projectDir))
}

// sanitizeProjectName normalizes a project name to be safe for file system use.
func sanitizeProjectName(name string) string {
	name = strings.ToLower(name)
	// Replace non-alphanumeric characters (except hyphens) with hyphens.
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
