package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Global  map[string]string
	Presets map[string]map[string]string
}

func ParseConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{
		Global:  make(map[string]string),
		Presets: make(map[string]map[string]string),
	}

	current := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = line[1 : len(line)-1]
			if current != "*" {
				cfg.Presets[current] = make(map[string]string)
			}
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if current == "*" || current == "" {
			cfg.Global[key] = val
		} else {
			cfg.Presets[current][key] = val
		}
	}
	return cfg, scanner.Err()
}

func (c *Config) ProxyPort() (int, error) {
	p, ok := c.Global["port"]
	if !ok {
		return 11434, nil
	}
	return strconv.Atoi(p)
}

func (c *Config) ProxyHost() string {
	if h, ok := c.Global["host"]; ok {
		return h
	}
	return "0.0.0.0"
}

func (c *Config) PresetNames() []string {
	names := make([]string, 0, len(c.Presets))
	for k := range c.Presets {
		names = append(names, k)
	}
	return names
}

// boolFlags are llama-server flags emitted without a value argument.
var boolFlags = map[string]bool{
	"jinja": true,
}

// LlamaServerArgs builds CLI args from global settings, overriding host/port for internal use.
func (c *Config) LlamaServerArgs(internalPort int) []string {
	var args []string
	for k, v := range c.Global {
		switch k {
		case "host":
			args = append(args, "--host", "127.0.0.1")
		case "port":
			args = append(args, "--port", fmt.Sprintf("%d", internalPort))
		default:
			if boolFlags[k] && (v == "true" || v == "1") {
				args = append(args, "--"+k)
			} else {
				args = append(args, "--"+k, v)
			}
		}
	}
	return args
}
