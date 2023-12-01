package config

import (
	"bytes"
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
	Env            map[string]string  `json:"env,omitempty"`
	DefaultProfile string             `json:"defaultProfile"`
	Profiles       map[string]Profile `json:"profiles"`
}

type Profile struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
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
var Path string = FindConfigPath()
var schema *jsonschema.Schema

func init() {
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	compiler.AddResource("schema.json", bytes.NewReader(schemaBytes))
	schema = compiler.MustCompile("schema.json")
}

func FindConfigPath() string {
	if env, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		if _, err := os.Stat(filepath.Join(env, "popcorn", "popcorn.jsonc")); err == nil {
			return filepath.Join(env, "popcorn", "popcorn.jsonc")
		}

		if _, err := os.Stat(filepath.Join(env, "popcorn", "popcorn.json")); err == nil {
			return filepath.Join(env, "popcorn", "popcorn.json")
		}
	}

	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".config", "popcorn", "popcorn.jsonc")); err == nil {
		return filepath.Join(os.Getenv("HOME"), ".config", "popcorn", "popcorn.jsonc")
	}

	return filepath.Join(os.Getenv("HOME"), ".config", "popcorn", "popcorn.json")
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

func Load(Path string) (Config, error) {
	configBytes, err := os.ReadFile(Path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(Path), 0755); err != nil {
			return Config{}, err
		}

		jsonBytes, err := json.MarshalIndent(DefaultConfig, "", "  ")
		if err != nil {
			return Config{}, err
		}

		if err := os.WriteFile(Path, jsonBytes, 0644); err != nil {
			return Config{}, err
		}

		return DefaultConfig, nil
	} else if err != nil {
		return Config{}, err
	}

	if filepath.Ext(Path) == ".jsonc" {
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
