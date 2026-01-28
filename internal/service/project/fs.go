package project

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func readJSON(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func writeJSONAtomic(path string, v interface{}) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func requireNonEmptyString(value string, message string) error {
	if value == "" {
		return errors.New(message)
	}
	return nil
}

