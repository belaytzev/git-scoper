package main

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Name  string
	Email string
}

// parseKeyValue reads a simple key=value config file (name= and email= lines).
func parseKeyValue(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config file not found at %s: %w", path, err)
	}
	cfg := &Config{}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "name":
			cfg.Name = strings.TrimSpace(v)
		case "email":
			cfg.Email = strings.TrimSpace(v)
		}
	}
	if cfg.Name == "" || cfg.Email == "" {
		return nil, fmt.Errorf("%s must contain both name= and email=", path)
	}
	return cfg, nil
}
