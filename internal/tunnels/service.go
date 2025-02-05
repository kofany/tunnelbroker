package tunnels

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/kofany/tunnelbroker/internal/config"
)

// TunnelCommands zawiera listę komend systemowych dla konfiguracji tunelu.
type TunnelCommands struct {
	Server []string `json:"server"`
	Client []string `json:"client"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// parsePrefix parses IPv6 prefix and returns net.IPNet object
func parsePrefix(prefix string) (*net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return nil, fmt.Errorf("invalid IPv6 prefix: %w", err)
	}
	return ipNet, nil
}

// generateDelegatedPrefix generates a delegated /64 prefix from a larger prefix
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

	// Set randomBit in third tercet (index 5)
	newIP[5] = byte(randomBit)

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

	// Generate random bit (0 or 1)
	randomBit := rand.Intn(2)

	// Generate delegated prefixes
	delegatedPrefix1, err := generateDelegatedPrefix(primaryPrefix, randomBit, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("error generating first prefix: %w", err)
	}

	delegatedPrefix2, err := generateDelegatedPrefix(secondaryPrefix, randomBit, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("error generating second prefix: %w", err)
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
		}
		commands.Client = []string{
			fmt.Sprintf("ip tunnel add %s mode sit local %s remote %s ttl 255", tunnelID, clientIPv4, serverIPv4),
			fmt.Sprintf("ip link set %s up", tunnelID),
			fmt.Sprintf("ip -6 addr add %s dev %s", endpointRemote, tunnelID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix1, "/64"), tunnelID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix2, "/64"), tunnelID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(endpointLocal, "/64"), tunnelID),
		}
	} else if strings.ToLower(tunnelType) == "gre" {
		commands.Server = []string{
			fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", tunnelID, serverIPv4, clientIPv4),
			fmt.Sprintf("ip link set %s up", tunnelID),
			fmt.Sprintf("ip -6 addr add %s dev %s", endpointLocal, tunnelID),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix1, tunnelID),
			fmt.Sprintf("ip -6 route add %s dev %s", delegatedPrefix2, tunnelID),
		}
		commands.Client = []string{
			fmt.Sprintf("ip tunnel add %s mode gre local %s remote %s ttl 255", tunnelID, clientIPv4, serverIPv4),
			fmt.Sprintf("ip link set %s up", tunnelID),
			fmt.Sprintf("ip -6 addr add %s dev %s", endpointRemote, tunnelID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix1, "/64"), tunnelID),
			fmt.Sprintf("ip -6 addr add %s1/64 dev %s", strings.TrimSuffix(delegatedPrefix2, "/64"), tunnelID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(endpointLocal, "/64"), tunnelID),
		}
	} else {
		return nil, nil, errors.New("invalid tunnel type")
	}

	return tunnel, commands, nil
}
