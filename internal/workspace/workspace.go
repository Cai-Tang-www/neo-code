package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manager constrains file operations to a single workspace root.
type Manager struct {
	root string
}

func New(root string) (*Manager, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}

	return &Manager{root: filepath.Clean(absRoot)}, nil
}

func (m *Manager) Root() string {
	return m.root
}

func (m *Manager) Resolve(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is required")
	}

	candidate := path
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(m.root, candidate)
	}

	candidate = filepath.Clean(candidate)
	rel, err := filepath.Rel(m.root, candidate)
	if err != nil {
		return "", fmt.Errorf("resolve relative path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace root %q", path, m.root)
	}

	return candidate, nil
}

func (m *Manager) ReadFile(path string) (string, error) {
	resolved, err := m.Resolve(path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", path, err)
	}
	return string(data), nil
}

func (m *Manager) WriteFile(path, content string) error {
	resolved, err := m.Resolve(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %s: %w", path, err)
	}

	if err := os.WriteFile(resolved, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}

func (m *Manager) ReplaceInFile(path, old, new string) (string, error) {
	if old == "" {
		return "", fmt.Errorf("old text must not be empty")
	}

	content, err := m.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !strings.Contains(content, old) {
		return "", fmt.Errorf("target text not found in %s", path)
	}

	updated := strings.Replace(content, old, new, 1)
	if err := m.WriteFile(path, updated); err != nil {
		return "", err
	}

	return updated, nil
}
