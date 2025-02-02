package state

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kofany/tunnelbroker/internal/interfaces"
)

type PrefixPool struct {
	Pools []struct {
		BasePrefix    string
		EndpointBlock string
		UsedPrefixes  map[string]bool
	}
	CurrentPair int
}

func NewPrefixPool() *PrefixPool {
	return &PrefixPool{
		Pools: []struct {
			BasePrefix    string
			EndpointBlock string
			UsedPrefixes  map[string]bool
		}{
			{"2a05:dfc3:ff00::/40", "2a05:dfc3:ffff::/48", make(map[string]bool)},
			{"2a05:1083::/32", "2a05:1083:ffff::/48", make(map[string]bool)},
			{"2a12:bec0:2c0::/44", "2a12:bec0:2cf::/48", make(map[string]bool)},
			{"2a05:dfc1:3c00::/40", "2a05:dfc1:3cff::/48", make(map[string]bool)},
		},
	}
}

func (p *PrefixPool) generateRandomPrefix(basePrefix string, db interfaces.Database) (string, error) {
	maxAttempts := 1000
	attempts := 0

	for attempts < maxAttempts {
		rand := rand.New(rand.NewSource(time.Now().UnixNano()))

		// Losujemy numer /48 (omijając ffff zarezerwowane dla endpointów)
		subnet48 := rand.Intn(0xfffe)

		// Losujemy numer /64 w tym /48
		subnet64 := rand.Intn(0xffff)

		prefix := fmt.Sprintf("%s:%04x:%04x::/64", basePrefix[:len(basePrefix)-7], subnet48, subnet64)

		// Sprawdzamy w pamięci podręcznej
		if !p.Pools[p.CurrentPair*2].UsedPrefixes[prefix] {
			// Sprawdzamy w bazie danych
			inUse, err := db.IsPrefixInUse(prefix)
			if err != nil {
				return "", fmt.Errorf("failed to check prefix usage: %v", err)
			}

			if !inUse {
				p.Pools[p.CurrentPair*2].UsedPrefixes[prefix] = true
				return prefix, nil
			}
		}

		attempts++
	}

	return "", fmt.Errorf("failed to find free prefix after %d attempts", maxAttempts)
}

func (p *PrefixPool) AllocateForTunnel(db interfaces.Database) (endpointLocal, endpointRemote, prefix1, prefix2 string, err error) {
	startIdx := (p.CurrentPair * 2) % len(p.Pools)

	// Endpointy z pierwszej klasy z pari (zawsze z ffff:1)
	endpointLocal = fmt.Sprintf("%s:ffff:1::1/64", p.Pools[startIdx].BasePrefix[:len(p.Pools[startIdx].BasePrefix)-7])
	endpointRemote = fmt.Sprintf("%s:ffff:1::2/64", p.Pools[startIdx].BasePrefix[:len(p.Pools[startIdx].BasePrefix)-7])

	// Generujemy pierwszy prefiks
	prefix1, err = p.generateRandomPrefix(p.Pools[startIdx].BasePrefix, db)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to allocate first prefix: %v", err)
	}

	// Generujemy drugi prefiks
	prefix2, err = p.generateRandomPrefix(p.Pools[startIdx+1].BasePrefix, db)
	if err != nil {
		// Wycofujemy alokację pierwszego prefiksu
		delete(p.Pools[startIdx].UsedPrefixes, prefix1)
		return "", "", "", "", fmt.Errorf("failed to allocate second prefix: %v", err)
	}

	// Przełącz na następną parę pul
	p.CurrentPair = (p.CurrentPair + 1) % 2

	return endpointLocal, endpointRemote, prefix1, prefix2, nil
}

func (p *PrefixPool) ReleasePrefixes(prefix1, prefix2 string) {
	// Zwalniamy prefiksy z pamięci podręcznej
	for i := range p.Pools {
		delete(p.Pools[i].UsedPrefixes, prefix1)
		delete(p.Pools[i].UsedPrefixes, prefix2)
	}
}
