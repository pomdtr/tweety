package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/tailscale/hujson"
)

type Config struct {
	Theme          string             `json:"theme"`
	ThemeDark      string             `json:"themeDark"`
	XTerm          map[string]any     `json:"xterm,omitempty"`
	Env            map[string]string  `json:"env,omitempty"`
	DefaultProfile string             `json:"defaultProfile"`
	Profiles       map[string]Profile `json:"profiles"`
}

type Profile struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Cwd     string            `json:"cwd,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Favicon string            `json:"favicon,omitempty"`
}

var DefaultConfig = Config{
	Theme:          "Tomorrow",
	ThemeDark:      "Tomorrow Night",
	DefaultProfile: "default",
	Profiles: map[string]Profile{
		"default": {
			Command: defaultShell(),
			Args:    []string{"-l"},
		},
	},
}

var schemaBytes, _ = json.MarshalIndent(DefaultConfig, "", "  ")
var configPath string = FindConfigPath()
var schema *jsonschema.Schema

func init() {
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	compiler.AddResource("schema.json", bytes.NewReader(schemaBytes))
	schema = compiler.MustCompile("schema.json")
}

func FindConfigPath() string {
	if env, ok := os.LookupEnv("TWEETY_CONFIG"); ok {
		return env
	}
	if env, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		if _, err := os.Stat(filepath.Join(env, "tweety", "tweety.jsonc")); err == nil {
			return filepath.Join(env, "tweety", "tweety.jsonc")
		}

		if _, err := os.Stat(filepath.Join(env, "tweety", "tweety.json")); err == nil {
			return filepath.Join(env, "tweety", "tweety.json")
		}
	}

	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".config", "tweety", "tweety.jsonc")); err == nil {
		return filepath.Join(os.Getenv("HOME"), ".config", "tweety", "tweety.jsonc")
	}

	return filepath.Join(os.Getenv("HOME"), ".config", "tweety", "tweety.json")
}

func defaultShell() string {
	shell, ok := os.LookupEnv("SHELL")
	if ok {
		return shell
	}

	switch runtime.GOOS {
	case "darwin":
		return "/bin/zsh"
	default:
		return "/bin/sh"
	}
}

func LoadConfig(configPath string) (Config, error) {
	configBytes, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return Config{}, err
		}

		jsonBytes, err := json.MarshalIndent(DefaultConfig, "", "  ")
		if err != nil {
			return Config{}, err
		}

		if err := os.WriteFile(configPath, jsonBytes, 0644); err != nil {
			return Config{}, err
		}

		configBytes = jsonBytes
	} else if err != nil {
		return Config{}, err
	}

	if filepath.Ext(configPath) == ".jsonc" {
		jsonBytes, err := hujson.Standardize(configBytes)
		if err != nil {
			return Config{}, err
		}
		configBytes = jsonBytes
	}

	var v any
	if err := json.Unmarshal(configBytes, &v); err != nil {
		return Config{}, err
	}

	if err := schema.Validate(v); err != nil {
		return Config{}, err
	}

	var config Config
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}
