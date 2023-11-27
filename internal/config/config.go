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

var schemaBytes, _ = json.MarshalIndent(DefaultConfig, "", "  ")
var Path string
var schema *jsonschema.Schema

func init() {
	if env, ok := os.LookupEnv("POPCORN_CONFIG"); ok {
		Path = env
	} else if env, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		Path = filepath.Join(env, "popcorn", "popcorn.json")
	} else {
		Path = filepath.Join(os.Getenv("HOME"), ".config", "popcorn", "popcorn.json")
	}

	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	compiler.AddResource("schema.json", bytes.NewReader(schemaBytes))
	schema = compiler.MustCompile("schema.json")
}

type Config struct {
	Profiles map[string]Profile `json:"profiles"`
}

type Profile struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
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

var DefaultConfig = Config{
	Profiles: map[string]Profile{
		"default": {
			Command: defaultShell(),
			Args:    []string{"-li"},
		},
	},
}

func Load(Path string) (Config, error) {
	configBytes, err := os.ReadFile(Path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultConfig, nil
	} else if err != nil {
		return Config{}, err
	}

	jsonBytes, err := hujson.Standardize(configBytes)
	if err != nil {
		return Config{}, err
	}

	var v any
	if err := json.Unmarshal(jsonBytes, &v); err != nil {
		return Config{}, err
	}

	if err := schema.Validate(v); err != nil {
		return Config{}, err
	}

	var config Config
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}
