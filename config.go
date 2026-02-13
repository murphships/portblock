package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Config represents .portblock.yaml configuration
type Config struct {
	Port   int    `yaml:"port" json:"port"`
	Seed   int64  `yaml:"seed" json:"seed"`
	Delay  string `yaml:"delay" json:"delay"`
	Chaos  bool   `yaml:"chaos" json:"chaos"`
	NoAuth bool   `yaml:"no-auth" json:"no-auth"`
	Watch  *bool  `yaml:"watch" json:"watch"`
	Strict bool   `yaml:"strict" json:"strict"`

	WebhookTarget string `yaml:"webhook-target" json:"webhook-target"`
	WebhookDelay  string `yaml:"webhook-delay" json:"webhook-delay"`
}

func loadConfig() *Config {
	// check CWD first, then home dir
	locations := []string{}

	cwd, err := os.Getwd()
	if err == nil {
		locations = append(locations,
			filepath.Join(cwd, ".portblock.yaml"),
			filepath.Join(cwd, ".portblock.yml"),
			filepath.Join(cwd, ".portblock.json"),
		)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		locations = append(locations,
			filepath.Join(home, ".portblock.yaml"),
			filepath.Join(home, ".portblock.yml"),
			filepath.Join(home, ".portblock.json"),
		)
	}

	for _, path := range locations {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		cfg := &Config{}
		ext := filepath.Ext(path)
		switch ext {
		case ".json":
			if err := json.Unmarshal(data, cfg); err != nil {
				continue
			}
		default:
			if err := yaml.Unmarshal(data, cfg); err != nil {
				continue
			}
		}
		return cfg
	}

	return nil
}

func applyConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.Port != 0 && port == 4000 { // only apply if still default
		port = cfg.Port
	}
	if cfg.Seed != 0 && seed == 0 {
		seed = cfg.Seed
	}
	if cfg.Delay != "" && delay == 0 {
		if d, err := time.ParseDuration(cfg.Delay); err == nil {
			delay = d
		}
	}
	if cfg.Chaos && !chaos {
		chaos = true
	}
	if cfg.NoAuth && !noAuth {
		noAuth = true
	}
	if cfg.Strict && !strictMode {
		strictMode = true
	}
	if cfg.WebhookTarget != "" && webhookTarget == "" {
		webhookTarget = cfg.WebhookTarget
	}
	if cfg.WebhookDelay != "" && webhookDelay == 0 {
		if d, err := time.ParseDuration(cfg.WebhookDelay); err == nil {
			webhookDelay = d
		}
	}
}

func applyConfigWatch(cfg *Config, cmd *cobra.Command) {
	if cfg == nil {
		return
	}
	if cfg.Watch != nil && !cmd.Flags().Changed("watch") {
		cmd.Flags().Set("watch", fmt.Sprintf("%v", *cfg.Watch))
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	defaultWatch := true
	cfg := Config{
		Port:   4000,
		Seed:   0,
		Delay:  "",
		Chaos:  false,
		NoAuth: false,
		Watch:  &defaultWatch,
		Strict: false,
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	filename := ".portblock.yaml"
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("%s already exists", filename)
	}

	header := "# portblock config — CLI flags override these values\n# docs: https://github.com/murphships/portblock\n\n"
	if err := os.WriteFile(filename, []byte(header+string(data)), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("created %s\n", filename)
	return nil
}
