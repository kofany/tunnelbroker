package tunnels

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"

	"github.com/kofany/tunnelbroker/internal/config"
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
	randomHex := (rand.Intn(8) + randomBit) % 16 // 0-15 dla hex

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
	ones, bits := ipNet.Mask.Size()
	if ones != 48 {
		return "", fmt.Errorf("base prefix must be /48, got /%d", ones)
	}

	// Convert userID to number
	userNum, err := strconv.ParseUint(userID, 16, 16)
	if err != nil {
		return "", fmt.Errorf("invalid userID: %w", err)
	}

	// Calculate new prefix
	newIP := make(net.IP, len(ipNet.IP))
	copy(newIP, ipNet.IP)

	// Set userID in fifth tercet (indices 8-9)
	newIP[8] = byte(userNum >> 8)
	newIP[9] = byte(userNum)

	// Create new /64 mask
	mask := net.CIDRMask(64, bits)

	newNet := net.IPNet{
		IP:   newIP,
		Mask: mask,
	}
	return newNet.String(), nil
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
		randomBit := rand.Intn(2)
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
		randomBit := rand.Intn(2)
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

	seq := uint64(rand.Intn(0x10000))
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
	thirdPrefix := config.GlobalConfig.Prefixes.Third
	// Remove mask from third prefix
	thirdPrefix = strings.TrimSuffix(thirdPrefix, "/48")

	// Generate third prefix and check for uniqueness
	var delegatedPrefix3 string
	for range maxAttempts {
		tempPrefix, err := generateThirdPrefix(thirdPrefix, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("error generating third prefix: %w", err)
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

	// If we couldn't find a unique prefix after maxAttempts
	if delegatedPrefix3 == "" {
		return nil, nil, fmt.Errorf("could not generate a unique third prefix after %d attempts", maxAttempts)
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
	} else {
		return nil, nil, errors.New("invalid tunnel type")
	}

	return tunnel, commands, nil
}
