package tunnels

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	mathrand "math/rand"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kofany/tunnelbroker/internal/config"
	"golang.org/x/crypto/curve25519"
)

// TunnelCommands zawiera listę komend systemowych dla konfiguracji tunelu.
type TunnelCommands struct {
	Server []string `json:"server"`
	Client []string `json:"client"`
}

// No need to seed the random number generator in Go 1.20+
// The random number generator is automatically seeded

// parsePrefix parses IPv6 prefix and returns net.IPNet object
func parsePrefix(prefix string) (*net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return nil, fmt.Errorf("invalid IPv6 prefix: %w", err)
	}
	return ipNet, nil
}

// generateDelegatedPrefix generates a delegated /64 prefix from a larger prefix
// randomBit is used to add entropy to the generated prefix
func generateDelegatedPrefix(basePrefix string, randomBit int, userID string) (string, error) {
	// Add /44 mask if not present
	if !strings.Contains(basePrefix, "/") {
		basePrefix = basePrefix + "/44"
	}

	// Parse base prefix
	ipNet, err := parsePrefix(basePrefix)
	if err != nil {
		return "", err
	}

	// Check if prefix is /44 (required for our logic)
	ones, bits := ipNet.Mask.Size()
	if ones != 44 {
		return "", fmt.Errorf("base prefix must be /44, got /%d", ones)
	}

	// Convert userID to number
	userNum, err := strconv.ParseUint(userID, 16, 16)
	if err != nil {
		return "", fmt.Errorf("invalid userID: %w", err)
	}

	// Calculate new prefix
	newIP := make(net.IP, len(ipNet.IP))
	copy(newIP, ipNet.IP)

	// Use randomBit to influence the generated hex for last position in third tercet (0-F)
	// This adds more entropy to the prefix generation
	randomHex := (mathrand.Intn(8) + randomBit) % 16 // 0-15 dla hex

	// Set random hex in last position of third tercet (index 5)
	newIP[5] = newIP[5]&0xF0 | byte(randomHex) // Zachowuj pierwsze 4 bity, ustaw ostatnie 4

	// Set userID in fourth tercet (indices 6-7)
	newIP[6] = byte(userNum >> 8)
	newIP[7] = byte(userNum)

	// Create new /64 mask
	mask := net.CIDRMask(64, bits)

	newNet := net.IPNet{
		IP:   newIP,
		Mask: mask,
	}
	return newNet.String(), nil
}

// generateThirdPrefix generates a delegated /64 prefix from a /48 prefix
// Format: {/48_prefix}:{user_id}::/64
func generateThirdPrefix(basePrefix string, userID string) (string, error) {
	// Add /48 mask if not present
	if !strings.Contains(basePrefix, "/") {
		basePrefix = basePrefix + "/48"
	}

	// Parse base prefix
	ipNet, err := parsePrefix(basePrefix)
	if err != nil {
		return "", err
	}

	// Check if prefix is /48 (required for our logic)
	ones, _ := ipNet.Mask.Size()
	if ones != 48 {
		return "", fmt.Errorf("base prefix must be /48, got /%d", ones)
	}

	// Validate that userID is a valid hex number
	_, err = strconv.ParseUint(userID, 16, 16)
	if err != nil {
		return "", fmt.Errorf("invalid userID: %w", err)
	}

	// Extract the base prefix without the mask and remove the last segment
	parts := strings.Split(strings.Split(basePrefix, "/")[0], ":")
	if len(parts) >= 4 {
		// Take only the first 3 segments
		basePrefixParts := parts[:3]
		basePrefixWithoutMask := strings.Join(basePrefixParts, ":")

		// Format: 2a03:94e0:2496:7696::/64
		result := fmt.Sprintf("%s:%s::/64", basePrefixWithoutMask, userID)
		return result, nil
	}

	return "", fmt.Errorf("invalid prefix format: %s", basePrefix)
}

// validateIPv6Address checks if IPv6 address is valid
func validateIPv6Address(address string) error {
	// Remove CIDR mask if present
	ipStr := strings.Split(address, "/")[0]

	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() != nil {
		return fmt.Errorf("invalid IPv6 address: %s", address)
	}
	return nil
}

// WireGuardKeyPair represents a WireGuard private/public key pair
type WireGuardKeyPair struct {
	PrivateKey string
	PublicKey  string
}

// generateWireGuardKeyPair generates a new WireGuard private/public key pair
// Uses curve25519 cryptography as required by WireGuard protocol
func generateWireGuardKeyPair() (*WireGuardKeyPair, error) {
	// Generate 32 random bytes for private key
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Clamp the private key as required by Curve25519
	// This ensures the key is in the correct format
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	// Derive the public key from the private key using Curve25519
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	// Encode keys to base64 (WireGuard standard format)
	keyPair := &WireGuardKeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(privateKey[:]),
		PublicKey:  base64.StdEncoding.EncodeToString(publicKey[:]),
	}

	return keyPair, nil
}

// generateWireGuardKeyPairWithCommand generates keys using wg command (alternative method)
// This is a fallback method if the native Go implementation doesn't work
func generateWireGuardKeyPairWithCommand() (*WireGuardKeyPair, error) {
	// Generate private key
	cmd := exec.Command("wg", "genkey")
	var privateKeyOut bytes.Buffer
	cmd.Stdout = &privateKeyOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to generate private key with wg command: %w", err)
	}
	privateKey := strings.TrimSpace(privateKeyOut.String())

	// Generate public key from private key
	cmd = exec.Command("wg", "pubkey")
	cmd.Stdin = strings.NewReader(privateKey)
	var publicKeyOut bytes.Buffer
	cmd.Stdout = &publicKeyOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to generate public key with wg command: %w", err)
	}
	publicKey := strings.TrimSpace(publicKeyOut.String())

	return &WireGuardKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// CreateTunnelService implements business logic for tunnel creation:
// - Checks user tunnel limit (max 2)
// - Generates tunnel ID, assigns prefixes and endpoints
// - Inserts record to database and generates system commands
func CreateTunnelService(tunnelType, userID, clientIPv4, serverIPv4 string) (*Tunnel, *TunnelCommands, error) {
	// Check number of active tunnels
	count, err := CountActiveTunnelsByUser(userID)
	if err != nil {
		return nil, nil, err
	}
	if count >= 2 {
		return nil, nil, errors.New("limit of 2 active tunnels per user has been reached")
	}
	pairNumber := 1
	if count == 1 {
		pairNumber = 2
	}

	tunnelID := fmt.Sprintf("tun-%s-%d", userID, pairNumber)

	// Get prefixes from configuration
	var primaryPrefix, secondaryPrefix string
	if pairNumber == 1 {
		primaryPrefix = config.GlobalConfig.Prefixes.Para1.Primary
		secondaryPrefix = config.GlobalConfig.Prefixes.Para1.Secondary
	} else {
		primaryPrefix = config.GlobalConfig.Prefixes.Para2.Primary
		secondaryPrefix = config.GlobalConfig.Prefixes.Para2.Secondary
	}

	// Remove mask from prefixes
	primaryPrefix = strings.TrimSuffix(primaryPrefix, "/44")
	secondaryPrefix = strings.TrimSuffix(secondaryPrefix, "/44")

	// Generate delegated prefixes with uniqueness check
	// Define max attempts for all prefix generation
	const maxAttempts = 10 // Limit the number of attempts to avoid infinite loop

	// First prefix
	var delegatedPrefix1 string
	for range maxAttempts {
		// Generate random bit (0 or 1) for each attempt
		randomBit := mathrand.Intn(2)
		tempPrefix, err := generateDelegatedPrefix(primaryPrefix, randomBit, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("error generating first prefix: %w", err)
		}

		// Check if the prefix is already in use
		inUse, err := IsPrefixInUse(tempPrefix)
		if err != nil {
			return nil, nil, fmt.Errorf("error checking first prefix uniqueness: %w", err)
		}

		if !inUse {
			delegatedPrefix1 = tempPrefix
			break
		}
	}

	// If we couldn't find a unique prefix after maxAttempts
	if delegatedPrefix1 == "" {
		return nil, nil, fmt.Errorf("could not generate a unique first prefix after %d attempts", maxAttempts)
	}

	// Second prefix
	var delegatedPrefix2 string
	for range maxAttempts {
		// Generate random bit (0 or 1) for each attempt
		randomBit := mathrand.Intn(2)
		tempPrefix, err := generateDelegatedPrefix(secondaryPrefix, randomBit, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("error generating second prefix: %w", err)
		}

		// Check if the prefix is already in use
		inUse, err := IsPrefixInUse(tempPrefix)
		if err != nil {
			return nil, nil, fmt.Errorf("error checking second prefix uniqueness: %w", err)
		}

		if !inUse {
			delegatedPrefix2 = tempPrefix
			break
		}
	}

	// If we couldn't find a unique prefix after maxAttempts
	if delegatedPrefix2 == "" {
		return nil, nil, fmt.Errorf("could not generate a unique second prefix after %d attempts", maxAttempts)
	}

	// Generate ULA addresses for endpoints
	ulaBase, err := parsePrefix(config.GlobalConfig.Prefixes.ULA)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing ULA prefix: %w", err)
	}

	seq := uint64(mathrand.Intn(0x10000))
	ulaNet := &net.IPNet{
		IP:   make(net.IP, len(ulaBase.IP)),
		Mask: net.CIDRMask(64, 128),
	}
	copy(ulaNet.IP, ulaBase.IP)

	// Set sequence in appropriate bytes
	ulaNet.IP[6] = byte(seq >> 8)
	ulaNet.IP[7] = byte(seq)

	endpointLocal := fmt.Sprintf("%s1/64", strings.TrimSuffix(ulaNet.String(), "/64"))
	endpointRemote := fmt.Sprintf("%s2/64", strings.TrimSuffix(ulaNet.String(), "/64"))

	// Generate third prefix from dedicated /48
	// For the first tunnel, try to use the primary pool
	// For the second tunnel or if primary pool prefix is already in use, use the alternative pool
	var delegatedPrefix3 string

	// Try with primary pool first
	primaryThirdPrefix := config.GlobalConfig.Prefixes.Third
	// Remove mask from third prefix
	primaryThirdPrefix = strings.TrimSuffix(primaryThirdPrefix, "/48")

	// Check if we have an alternative pool configured
	altThirdPrefix := config.GlobalConfig.Prefixes.AltThird
	hasAltPool := altThirdPrefix != ""
	// Remove mask from alt third prefix if it exists
	if hasAltPool {
		altThirdPrefix = strings.TrimSuffix(altThirdPrefix, "/48")
	}

	// For first tunnel, try primary pool first
	if pairNumber == 1 {
		// Try with primary pool
		for range maxAttempts {
			tempPrefix, err := generateThirdPrefix(primaryThirdPrefix, userID)
			if err != nil {
				return nil, nil, fmt.Errorf("error generating third prefix from primary pool: %w", err)
			}

			// Check if the prefix is already in use
			inUse, err := IsPrefixInUse(tempPrefix)
			if err != nil {
				return nil, nil, fmt.Errorf("error checking prefix uniqueness: %w", err)
			}

			if !inUse {
				delegatedPrefix3 = tempPrefix
				break
			}
		}
	}

	// If we couldn't find a unique prefix in primary pool or this is the second tunnel,
	// and we have an alternative pool, try that
	if delegatedPrefix3 == "" && hasAltPool {
		for range maxAttempts {
			tempPrefix, err := generateThirdPrefix(altThirdPrefix, userID)
			if err != nil {
				return nil, nil, fmt.Errorf("error generating third prefix from alternative pool: %w", err)
			}

			// Check if the prefix is already in use
			inUse, err := IsPrefixInUse(tempPrefix)
			if err != nil {
				return nil, nil, fmt.Errorf("error checking prefix uniqueness: %w", err)
			}

			if !inUse {
				delegatedPrefix3 = tempPrefix
				break
			}
		}
	}

	// If we still couldn't find a unique prefix, try again with primary pool as fallback
	// This handles the case where alt-third is not configured
	if delegatedPrefix3 == "" && (pairNumber != 1 || !hasAltPool) {
		for range maxAttempts {
			tempPrefix, err := generateThirdPrefix(primaryThirdPrefix, userID)
			if err != nil {
				return nil, nil, fmt.Errorf("error generating third prefix from primary pool (fallback): %w", err)
			}

			// Check if the prefix is already in use
			inUse, err := IsPrefixInUse(tempPrefix)
			if err != nil {
				return nil, nil, fmt.Errorf("error checking prefix uniqueness: %w", err)
			}

			if !inUse {
				delegatedPrefix3 = tempPrefix
				break
			}
		}
	}

	// If we couldn't find a unique prefix after all attempts
	if delegatedPrefix3 == "" {
		return nil, nil, fmt.Errorf("could not generate a unique third prefix after multiple attempts")
	}

	tunnel := &Tunnel{
		ID:               tunnelID,
		UserID:           userID,
		Type:             tunnelType,
		Status:           "active",
		ServerIPv4:       serverIPv4,
		ClientIPv4:       clientIPv4,
		EndpointLocal:    endpointLocal,
		EndpointRemote:   endpointRemote,
		DelegatedPrefix1: delegatedPrefix1,
		DelegatedPrefix2: delegatedPrefix2,
		DelegatedPrefix3: delegatedPrefix3,
	}

	// Generate WireGuard client keys if tunnel type is "wg"
	if strings.ToLower(tunnelType) == "wg" {
		// Generate client keys only - server uses global wg0 keypair from config
		clientKeys, err := generateWireGuardKeyPair()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate client WireGuard keys: %w", err)
		}

		// Set WireGuard client key fields
		// ServerPrivateKey and ServerPublicKey are no longer per-tunnel (global wg0)
		tunnel.ServerPublicKey = config.GlobalConfig.WireGuard.PublicKey
		tunnel.ClientPrivateKey = clientKeys.PrivateKey
		tunnel.ClientPublicKey = clientKeys.PublicKey
		// ListenPort is now global from config
		tunnel.ListenPort = config.GlobalConfig.WireGuard.ListenPort
	}

	// Validate IPv6 addresses
	if err := validateIPv6Address(strings.TrimSuffix(endpointLocal, "/64")); err != nil {
		return nil, nil, fmt.Errorf("error validating endpoint_local: %w", err)
	}
	if err := validateIPv6Address(strings.TrimSuffix(endpointRemote, "/64")); err != nil {
		return nil, nil, fmt.Errorf("error validating endpoint_remote: %w", err)
	}
	if err := validateIPv6Address(strings.TrimSuffix(delegatedPrefix1, "/64")); err != nil {
		return nil, nil, fmt.Errorf("error validating delegated_prefix_1: %w", err)
	}
	if err := validateIPv6Address(strings.TrimSuffix(delegatedPrefix2, "/64")); err != nil {
		return nil, nil, fmt.Errorf("error validating delegated_prefix_2: %w", err)
	}
	if err := validateIPv6Address(strings.TrimSuffix(delegatedPrefix3, "/64")); err != nil {
		return nil, nil, fmt.Errorf("error validating delegated_prefix_3: %w", err)
	}

	// Use transaction to create tunnel
	if err := CreateTunnelWithTransaction(tunnel); err != nil {
		return nil, nil, err
	}

	// Generate system commands
	commands := &TunnelCommands{}
	if strings.ToLower(tunnelType) == "sit" {
		commands.Server = []string{
			fmt.Sprintf("ip tunnel add %s mode sit local %s remote %s ttl 255", tunnelID, serverIPv4, clientIPv4),
			fmt.Sprintf("ip link set %s up", tunnelID),
			fmt.Sprintf("ip -6 addr add %s dev %s", endpointLocal, tunnelID),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix1, tunnelID),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix2, tunnelID),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix3, tunnelID),
		}
		commands.Client = []string{
			fmt.Sprintf("ip tunnel add %s mode sit local %s remote %s ttl 255", tunnelID, clientIPv4, serverIPv4),
			fmt.Sprintf("ip link set %s up", tunnelID),
			fmt.Sprintf("ip -6 addr add %s dev %s", endpointRemote, tunnelID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix1, "/64"), tunnelID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix2, "/64"), tunnelID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix3, "/64"), tunnelID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(endpointLocal, "/64"), tunnelID),
		}
	} else if strings.ToLower(tunnelType) == "gre" {
		commands.Server = []string{
			fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", tunnelID, serverIPv4, clientIPv4),
			fmt.Sprintf("ip link set %s up", tunnelID),
			fmt.Sprintf("ip -6 addr add %s dev %s", endpointLocal, tunnelID),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix1, tunnelID),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix2, tunnelID),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix3, tunnelID),
		}
		commands.Client = []string{
			fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", tunnelID, clientIPv4, serverIPv4),
			fmt.Sprintf("ip link set %s up", tunnelID),
			fmt.Sprintf("ip -6 addr add %s dev %s", endpointRemote, tunnelID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix1, "/64"), tunnelID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix2, "/64"), tunnelID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix3, "/64"), tunnelID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(endpointLocal, "/64"), tunnelID),
		}
	} else if strings.ToLower(tunnelType) == "wg" {
		// WireGuard configuration commands - using single wg0 interface with peers
		wgInterface := config.GlobalConfig.WireGuard.Interface
		wgPort := config.GlobalConfig.WireGuard.ListenPort
		serverPubKey := config.GlobalConfig.WireGuard.PublicKey

		// Build allowed-ips list including all prefixes
		allowedIPs := fmt.Sprintf("%s,%s,%s,%s", endpointRemote, delegatedPrefix1, delegatedPrefix2, delegatedPrefix3)

		// Server-side: add peer to wg0 and routes
		commands.Server = []string{
			fmt.Sprintf("wg set %s peer %s allowed-ips %s", wgInterface, tunnel.ClientPublicKey, allowedIPs),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix1, wgInterface),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix2, wgInterface),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix3, wgInterface),
		}

		// Client-side commands (for user to run)
		clientInterface := tunnelID
		commands.Client = []string{
			fmt.Sprintf("ip link add dev %s type wireguard", clientInterface),
			fmt.Sprintf("ip -6 addr add %s dev %s", endpointRemote, clientInterface),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix1, "/64"), clientInterface),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix2, "/64"), clientInterface),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix3, "/64"), clientInterface),
			fmt.Sprintf("echo '%s' > /etc/wireguard/%s_private.key && chmod 600 /etc/wireguard/%s_private.key",
				tunnel.ClientPrivateKey, clientInterface, clientInterface),
			fmt.Sprintf("wg set %s private-key /etc/wireguard/%s_private.key peer %s endpoint %s:%d allowed-ips ::/0",
				clientInterface, clientInterface, serverPubKey, serverIPv4, wgPort),
			fmt.Sprintf("ip link set %s up", clientInterface),
			fmt.Sprintf("ip -6 route add ::/0 dev %s", clientInterface),
		}
	} else {
		return nil, nil, errors.New("invalid tunnel type")
	}

	return tunnel, commands, nil
}
