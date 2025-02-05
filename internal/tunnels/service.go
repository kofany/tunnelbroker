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

// parsePrefix parsuje prefix IPv6 i zwraca obiekt net.IPNet
func parsePrefix(prefix string) (*net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return nil, fmt.Errorf("nieprawidłowy prefix IPv6: %w", err)
	}
	return ipNet, nil
}

// generateDelegatedPrefix generuje delegowany prefix /64 z większego prefixu
func generateDelegatedPrefix(basePrefix string, randomBit int, userID string) (string, error) {
	// Dodaj maskę /44 jeśli jej nie ma
	if !strings.Contains(basePrefix, "/") {
		basePrefix = basePrefix + "/44"
	}

	// Parsuj bazowy prefix
	ipNet, err := parsePrefix(basePrefix)
	if err != nil {
		return "", err
	}

	// Sprawdź czy prefix jest /44 (wymagane dla naszej logiki)
	ones, bits := ipNet.Mask.Size()
	if ones != 44 {
		return "", fmt.Errorf("prefix bazowy musi być /44, otrzymano /%d", ones)
	}

	// Konwertuj userID na liczbę
	userNum, err := strconv.ParseUint(userID, 16, 16)
	if err != nil {
		return "", fmt.Errorf("nieprawidłowy userID: %w", err)
	}

	// Oblicz nowy prefix
	newIP := make(net.IP, len(ipNet.IP))
	copy(newIP, ipNet.IP)

	// Ustaw randomBit w trzecim tercecie (indeks 5)
	newIP[5] = byte(randomBit)

	// Ustaw userID w czwartym tercecie (indeksy 6-7)
	newIP[6] = byte(userNum >> 8)
	newIP[7] = byte(userNum)

	// Stwórz nową maskę /64
	mask := net.CIDRMask(64, bits)

	newNet := net.IPNet{
		IP:   newIP,
		Mask: mask,
	}
	return newNet.String(), nil
}

// validateIPv6Address sprawdza poprawność adresu IPv6
func validateIPv6Address(address string) error {
	// Usuń maskę CIDR jeśli istnieje
	ipStr := strings.Split(address, "/")[0]

	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() != nil {
		return fmt.Errorf("nieprawidłowy adres IPv6: %s", address)
	}
	return nil
}

// CreateTunnelService wykonuje logikę biznesową tworzenia tunelu:
// - Sprawdza limit tuneli dla użytkownika (max 2)
// - Generuje ID tunelu, przydziela prefixy i endpointy
// - Wstawia rekord do bazy i generuje komendy systemowe
func CreateTunnelService(tunnelType, userID, clientIPv4, serverIPv4 string) (*Tunnel, *TunnelCommands, error) {
	// Sprawdzenie liczby aktywnych tuneli
	count, err := CountActiveTunnelsByUser(userID)
	if err != nil {
		return nil, nil, err
	}
	if count >= 2 {
		return nil, nil, errors.New("limit 2 aktywnych tuneli dla użytkownika został osiągnięty")
	}
	pairNumber := 1
	if count == 1 {
		pairNumber = 2
	}

	tunnelID := fmt.Sprintf("tun-%s-%d", userID, pairNumber)

	// Pobranie prefixów z konfiguracji
	var primaryPrefix, secondaryPrefix string
	if pairNumber == 1 {
		primaryPrefix = config.GlobalConfig.Prefixes.Para1.Primary
		secondaryPrefix = config.GlobalConfig.Prefixes.Para1.Secondary
	} else {
		primaryPrefix = config.GlobalConfig.Prefixes.Para2.Primary
		secondaryPrefix = config.GlobalConfig.Prefixes.Para2.Secondary
	}

	// Usunięcie maski z prefixów
	primaryPrefix = strings.TrimSuffix(primaryPrefix, "/44")
	secondaryPrefix = strings.TrimSuffix(secondaryPrefix, "/44")

	// Generowanie losowego bitu (0 lub 1)
	randomBit := rand.Intn(2)

	// Generowanie delegowanych prefixów
	delegatedPrefix1, err := generateDelegatedPrefix(primaryPrefix, randomBit, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("błąd generowania pierwszego prefixu: %w", err)
	}

	delegatedPrefix2, err := generateDelegatedPrefix(secondaryPrefix, randomBit, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("błąd generowania drugiego prefixu: %w", err)
	}

	// Generowanie adresów ULA dla endpointów
	ulaBase, err := parsePrefix(config.GlobalConfig.Prefixes.ULA)
	if err != nil {
		return nil, nil, fmt.Errorf("błąd parsowania prefixu ULA: %w", err)
	}

	seq := uint64(rand.Intn(0x10000))
	ulaNet := &net.IPNet{
		IP:   make(net.IP, len(ulaBase.IP)),
		Mask: net.CIDRMask(64, 128),
	}
	copy(ulaNet.IP, ulaBase.IP)

	// Ustaw sekwencję w odpowiednich bajtach
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

	// Walidacja adresów IPv6
	if err := validateIPv6Address(strings.TrimSuffix(endpointLocal, "/64")); err != nil {
		return nil, nil, fmt.Errorf("błąd walidacji endpoint_local: %w", err)
	}
	if err := validateIPv6Address(strings.TrimSuffix(endpointRemote, "/64")); err != nil {
		return nil, nil, fmt.Errorf("błąd walidacji endpoint_remote: %w", err)
	}
	if err := validateIPv6Address(strings.TrimSuffix(delegatedPrefix1, "/64")); err != nil {
		return nil, nil, fmt.Errorf("błąd walidacji delegated_prefix_1: %w", err)
	}
	if err := validateIPv6Address(strings.TrimSuffix(delegatedPrefix2, "/64")); err != nil {
		return nil, nil, fmt.Errorf("błąd walidacji delegated_prefix_2: %w", err)
	}

	// Użyj transakcji do utworzenia tunelu
	if err := CreateTunnelWithTransaction(tunnel); err != nil {
		return nil, nil, err
	}

	// Generowanie komend systemowych
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
			fmt.Sprintf("ip -6 addr add %s::1/64 dev %s", strings.TrimSuffix(delegatedPrefix1, "/64"), tunnelID),
			fmt.Sprintf("ip -6 addr add %s::1/64 dev %s", strings.TrimSuffix(delegatedPrefix2, "/64"), tunnelID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", strings.TrimSuffix(endpointLocal, "/64"), tunnelID),
		}
	} else {
		return nil, nil, errors.New("nieprawidłowy typ tunelu")
	}

	return tunnel, commands, nil
}
