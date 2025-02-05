package tunnels

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// TunnelCommands zawiera listę komend systemowych dla konfiguracji tunelu.
type TunnelCommands struct {
	Server []string `json:"server"`
	Client []string `json:"client"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
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

	// Definicja prefixów – hardkodujemy je na podstawie pary
	var primaryPrefix, secondaryPrefix string
	if pairNumber == 1 {
		primaryPrefix = "2a05:dfc1:3c00"
		secondaryPrefix = "2a12:bec0:2c00"
	} else {
		primaryPrefix = "2a05:1083:0000"
		secondaryPrefix = "2a05:dfc3:ff00"
	}

	// Generowanie losowego bitu (0 lub 1)
	randomBit := strconv.Itoa(rand.Intn(2))

	// Konstruowanie delegowanych prefixów: format {prefix}:{random_bit}:{user_id}::/64
	delegatedPrefix1 := fmt.Sprintf("%s:%s:%s::/64", primaryPrefix, randomBit, userID)
	delegatedPrefix2 := fmt.Sprintf("%s:%s:%s::/64", secondaryPrefix, randomBit, userID)

	// Generowanie adresów endpointów z prefiksu ULA: fde4:5a50:1114:{sequence}::{1,2}/64
	seq := fmt.Sprintf("%04x", rand.Intn(0x10000))
	endpointLocal := fmt.Sprintf("fde4:5a50:1114:%s::1/64", seq)
	endpointRemote := fmt.Sprintf("fde4:5a50:1114:%s::2/64", seq)

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

	// Wstawienie tunelu do bazy
	err = InsertTunnel(tunnel)
	if err != nil {
		return nil, nil, err
	}

	// Generowanie komend systemowych w zależności od typu tunelu
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
			// Dla uproszczenia wycinamy część "::" z delegowanych prefixów
			fmt.Sprintf("ip -6 addr add %s::1/64 dev %s", delegatedPrefix1[:strings.Index(delegatedPrefix1, "::")], tunnelID),
			fmt.Sprintf("ip -6 addr add %s::1/64 dev %s", delegatedPrefix2[:strings.Index(delegatedPrefix2, "::")], tunnelID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", endpointLocal, tunnelID),
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
			fmt.Sprintf("ip -6 addr add %s::1/64 dev %s", delegatedPrefix1[:strings.Index(delegatedPrefix1, "::")], tunnelID),
			fmt.Sprintf("ip -6 addr add %s::1/64 dev %s", delegatedPrefix2[:strings.Index(delegatedPrefix2, "::")], tunnelID),
			fmt.Sprintf("ip -6 route add ::/0 via %s dev %s", endpointLocal, tunnelID),
		}
	} else {
		return nil, nil, errors.New("nieprawidłowy typ tunelu")
	}

	return tunnel, commands, nil
}
