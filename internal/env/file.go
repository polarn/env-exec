package env

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

func WriteFiles(envVars map[string]string, names []string) (_ func() error, err error) {
	if len(names) == 0 {
		return func() error { return nil }, nil
	}

	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = os.TempDir()
	}
	dir, err := os.MkdirTemp(base, "env-exec-")
	if err != nil {
		return nil, fmt.Errorf("failed to create directory for file variables: %w", err)
	}
	remove := func() error {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to remove '%s': %w", dir, err)
		}
		return nil
	}
	defer func() {
		if err != nil {
			remove()
		}
	}()

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to open directory for file variables: %w", err)
	}
	defer root.Close()

	paths := make(map[string]string, len(names))
	for _, name := range names {
		value, ok := envVars[name]
		if !ok {
			continue
		}
		if err := root.WriteFile(name, []byte(value), 0600); err != nil {
			return nil, fmt.Errorf("failed to write file for '%s': %w", name, err)
		}
		paths[name] = filepath.Join(dir, name)
	}
	maps.Copy(envVars, paths)
	return remove, nil
}
