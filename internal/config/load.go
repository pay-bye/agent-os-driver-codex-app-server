package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type CodexCheck func(string) error

func Read(path string, check CodexCheck) (Config, error) {
	item, err := read(path)
	if err != nil {
		return Config{}, err
	}
	if err := item.Validate(check); err != nil {
		return Config{}, err
	}
	return item, nil
}

func read(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	var item Config
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&item); err != nil {
		return Config{}, fmt.Errorf("invalid_config_json: %w", err)
	}
	return item, nil
}
