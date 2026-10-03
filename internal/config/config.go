package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/charmbracelet/huh"
	clog "github.com/charmbracelet/log"
)

type Config struct {
	Dir  string `json:"dir"`
	Port int    `json:"port"`
}

func configPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	configDir = filepath.Join(configDir, "lantern")

	if err = os.MkdirAll(configDir, 0750); err != nil && !os.IsExist(err) {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		clog.Errorf("loadConfig: couldn't get config path: %s", err)
		return &Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			clog.Info("loadConfig: config doesn't exist")
			return &Config{}, nil
		}
		clog.Errorf("loadConfig: couldn't read config.json: %s", err)
		return &Config{}, err
	}

	var config Config
	return &config, json.Unmarshal(data, &config)
}

func SaveConfig(c Config) error {
	path, err := configPath()
	if err != nil {
		clog.Errorf("saveConfig: couldn't get config path: %s", err)
		return err
	}

	data, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		clog.Errorf("saveConfig: couldn't marshall config into json: %s", err)
		return err
	}

	return os.WriteFile(path, data, 0664)
}

func EnsureConfig() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, fmt.Errorf("get config path: %w", err)
	}

	if _, err = os.Stat(p); errors.Is(err, fs.ErrNotExist) {
		conf, err := interactiveConfigFill()
		if err != nil {
			return nil, fmt.Errorf("interactive fill: %w", err)
		}

		if err = SaveConfig(conf); err != nil {
			return nil, fmt.Errorf("save config to %s: %w", p, err)
		}
		clog.Infof("new config generated and saved at %s", p)
		return &conf, nil
	} else if err != nil {
		return nil, fmt.Errorf("config file check: %w", err)
	}
	return LoadConfig()
}

func interactiveConfigFill() (Config, error) {
	var dir, port string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Lantern Configuration").
				Description("Specify configuration parameters for config.json"),
			huh.NewInput().
				Title("Music Directory").
				Placeholder("./music").
				Value(&dir).
				Validate(func(s string) error {
					f, err := os.Stat(s)
					if err != nil {
						return errors.New("can't reach specified location")
					}
					if !f.IsDir() {
						return errors.New("not a directory")
					}
					return nil
				}),
			huh.NewInput().
				Title("Server Port").
				Placeholder("8080").
				Value(&port).
				Validate(func(s string) error {
					p, err := strconv.Atoi(s)
					if err != nil {
						return errors.New("port must be a number")
					}
					if p < 1 || p > 65535 {
						return errors.New("port must be within 1 and 65535")
					}
					return nil
				}),
		),
	)

	var conf Config
	err := form.Run()
	if err != nil {
		return conf, err
	}

	conf.Dir = dir
	conf.Port, _ = strconv.Atoi(port) // port is valid in form

	return conf, nil
}
