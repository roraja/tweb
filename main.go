package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	Shell    string `yaml:"shell"`
}

var (
	config       Config
	serverSecret string
)

func main() {
	configPath := flag.String("config", "", "path to config file (default: ~/tweb.yml)")
	port := flag.Int("port", 0, "port to listen on (overrides config)")
	flag.Parse()

	// Generate server secret for session signing
	secret := make([]byte, 32)
	rand.Read(secret)
	serverSecret = hex.EncodeToString(secret)

	if err := loadConfig(*configPath); err != nil {
		log.Fatalf("Config error: %v", err)
	}

	if *port > 0 {
		config.Port = *port
	}

	if config.Password == "" {
		log.Println("WARNING: No password set. Access is unrestricted.")
	}

	log.Printf("tweb starting on http://localhost:%d (shell: %s)", config.Port, config.Shell)
	startServer()
}

func loadConfig(path string) error {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot find home dir: %w", err)
		}
		path = filepath.Join(home, "tweb.yml")
	}

	config = Config{
		Port:  8080,
		Shell: defaultShell(),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("No config at %s, using defaults", path)
			return nil
		}
		return err
	}

	return yaml.Unmarshal(data, &config)
}

func defaultShell() string {
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}
