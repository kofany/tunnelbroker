package config

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

type PrefixPair struct {
	Primary   string `yaml:"primary"`
	Secondary string `yaml:"secondary"`
}

type PrefixConfig struct {
	Para1    PrefixPair `yaml:"para1"`
	Para2    PrefixPair `yaml:"para2"`
	ULA      string     `yaml:"ula"`
	Third    string     `yaml:"third"`
	AltThird string     `yaml:"alt-third"`
}

type ServerConfig struct {
	IPv4 string `yaml:"ipv4"`
}

type WireGuardConfig struct {
	Interface      string `yaml:"interface"`       // wg0
	ListenPort     int    `yaml:"listen_port"`     // 51820
	PrivateKeyFile string `yaml:"private_key_file"` // /etc/wireguard/server_private.key
	PublicKey      string `yaml:"public_key"`       // Server's public key (base64)
}

type APIConfig struct {
	Key    string `yaml:"key"`
	Listen string `yaml:"listen"`
}

type DatabaseConfig struct {
	MaxConnections     int    `yaml:"max_connections"`
	ConnectionLifetime string `yaml:"connection_lifetime"`
}

type Config struct {
	Prefixes  PrefixConfig    `yaml:"prefixes"`
	Server    ServerConfig    `yaml:"server"`
	API       APIConfig       `yaml:"api"`
	Database  DatabaseConfig  `yaml:"database"`
	WireGuard WireGuardConfig `yaml:"wireguard"`
}

var GlobalConfig Config

func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("błąd odczytu pliku konfiguracyjnego: %w", err)
	}

	if err := yaml.Unmarshal(data, &GlobalConfig); err != nil {
		return fmt.Errorf("błąd parsowania konfiguracji: %w", err)
	}

	// Load WireGuard public key if private key file exists
	if GlobalConfig.WireGuard.PrivateKeyFile != "" {
		if err := loadWireGuardPublicKey(); err != nil {
			// Log warning but don't fail - WireGuard might not be configured yet
			fmt.Printf("Warning: Could not load WireGuard public key: %v\n", err)
		}
	}

	return nil
}

// loadWireGuardPublicKey derives the public key from the private key file
func loadWireGuardPublicKey() error {
	privateKeyFile := GlobalConfig.WireGuard.PrivateKeyFile

	// Check if file exists
	if _, err := os.Stat(privateKeyFile); os.IsNotExist(err) {
		return fmt.Errorf("private key file does not exist: %s", privateKeyFile)
	}

	// Read private key
	privateKeyData, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return fmt.Errorf("failed to read private key file: %w", err)
	}

	privateKey := strings.TrimSpace(string(privateKeyData))

	// Derive public key using wg pubkey command
	cmd := exec.Command("wg", "pubkey")
	cmd.Stdin = strings.NewReader(privateKey)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to derive public key: %w", err)
	}

	GlobalConfig.WireGuard.PublicKey = strings.TrimSpace(out.String())
	return nil
}

// InitWireGuardInterface ensures wg0 interface exists and is configured
func InitWireGuardInterface() error {
	wgConfig := GlobalConfig.WireGuard
	if wgConfig.Interface == "" {
		return nil // WireGuard not configured
	}

	// Check if interface exists
	cmd := exec.Command("ip", "link", "show", wgConfig.Interface)
	if err := cmd.Run(); err != nil {
		// Interface doesn't exist, create it
		createCmd := exec.Command("ip", "link", "add", "dev", wgConfig.Interface, "type", "wireguard")
		if err := createCmd.Run(); err != nil {
			return fmt.Errorf("failed to create WireGuard interface: %w", err)
		}
	}

	// Configure listen port and private key
	setCmd := exec.Command("wg", "set", wgConfig.Interface,
		"listen-port", fmt.Sprintf("%d", wgConfig.ListenPort),
		"private-key", wgConfig.PrivateKeyFile)
	if err := setCmd.Run(); err != nil {
		return fmt.Errorf("failed to configure WireGuard interface: %w", err)
	}

	// Bring interface up
	upCmd := exec.Command("ip", "link", "set", wgConfig.Interface, "up")
	if err := upCmd.Run(); err != nil {
		return fmt.Errorf("failed to bring up WireGuard interface: %w", err)
	}

	return nil
}
