package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// SaveProjectEntry adds or updates a named project in the config file.
// Creates the config directory and file if they don't exist.
func SaveProjectEntry(name, pubKey, secretKey string) error {
	m, err := readConfigMap()
	if err != nil {
		return err
	}

	projects, _ := m["projects"].(map[string]interface{})
	if projects == nil {
		projects = make(map[string]interface{})
	}

	projects[name] = map[string]interface{}{
		"public_key": pubKey,
		"secret_key": secretKey,
	}
	m["projects"] = projects

	return writeConfigMap(m)
}

// RemoveProjectEntry removes a named project from the config file.
// If the removed project is the default_project, clears that setting too.
// Returns nil if the project is not found (idempotent).
func RemoveProjectEntry(name string) error {
	m, err := readConfigMap()
	if err != nil {
		return err
	}

	projects, _ := m["projects"].(map[string]interface{})
	if projects == nil {
		return nil
	}

	// YAML keys are case-sensitive, but try case-insensitive match
	// to be consistent with how Viper resolves project names.
	found := false
	for k := range projects {
		if strings.EqualFold(k, name) {
			delete(projects, k)
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	if len(projects) == 0 {
		delete(m, "projects")
	} else {
		m["projects"] = projects
	}

	// Clear default_project if it matches the removed project.
	if dp, _ := m["default_project"].(string); strings.EqualFold(dp, name) {
		delete(m, "default_project")
	}

	return writeConfigMap(m)
}

// RemoveProjectByPubKey finds and removes the project entry whose
// public_key matches pubKey. Also clears default_project if it pointed
// to the removed entry. Returns nil if no match is found (idempotent).
func RemoveProjectByPubKey(pubKey string) error {
	m, err := readConfigMap()
	if err != nil {
		return err
	}

	projects, _ := m["projects"].(map[string]interface{})
	if projects == nil {
		return nil
	}

	var removedName string
	for name, raw := range projects {
		entry, _ := raw.(map[string]interface{})
		if entry == nil {
			continue
		}
		if pk, _ := entry["public_key"].(string); pk == pubKey {
			delete(projects, name)
			removedName = name
			break
		}
	}
	if removedName == "" {
		return nil
	}

	if len(projects) == 0 {
		delete(m, "projects")
	} else {
		m["projects"] = projects
	}

	if dp, _ := m["default_project"].(string); strings.EqualFold(dp, removedName) {
		delete(m, "default_project")
	}

	return writeConfigMap(m)
}

// SetDefaultProject sets the default_project key in the config file.
func SetDefaultProject(name string) error {
	m, err := readConfigMap()
	if err != nil {
		return err
	}

	m["default_project"] = name
	return writeConfigMap(m)
}

// readConfigMap reads the config file as a generic map.
// Returns an empty map if the file does not exist.
func readConfigMap() (map[string]interface{}, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	m := make(map[string]interface{})
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return m, nil
}

// writeConfigMap writes a map back to the config file as YAML.
// Creates the config directory if it doesn't exist.
func writeConfigMap(m map[string]interface{}) error {
	dir := ConfigDir()
	if dir == "" {
		return fmt.Errorf("cannot determine config directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}
