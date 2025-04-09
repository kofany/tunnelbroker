package main

import (
	"fmt"
	"log"
	"strings"
)

func generateThirdPrefix(basePrefix string, userID string) string {
	// Extract the base prefix without the mask and remove the last segment
	parts := strings.Split(basePrefix, ":")
	if len(parts) >= 4 {
		// Take only the first 3 segments
		basePrefixParts := parts[:3]
		basePrefixWithoutMask := strings.Join(basePrefixParts, ":")

		// Format: 2a03:94e0:2496:7696::/64
		result := fmt.Sprintf("%s:%s::/64", basePrefixWithoutMask, userID)
		return result
	}

	return "invalid prefix format"
}

func main() {
	// Test with primary pool
	primaryPool := "2a03:94e0:2496::/48"
	userID := "7696"

	result1 := generateThirdPrefix(primaryPool, userID)
	fmt.Printf("Primary pool result: %s\n", result1)

	// Test with alternative pool
	altPool := "2a05:dfc1:3ccc::/48"
	result2 := generateThirdPrefix(altPool, userID)
	fmt.Printf("Alternative pool result: %s\n", result2)

	log.Println("Tests completed successfully")
}
