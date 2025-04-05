package tunnels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test for generateDelegatedPrefix function
func TestGenerateDelegatedPrefix(t *testing.T) {
	tests := []struct {
		name       string
		basePrefix string
		randomBit  int
		userID     string
		wantErr    bool
	}{
		{
			name:       "valid prefix generation",
			basePrefix: "2a05:dfc1:3c00::/44",
			randomBit:  0,
			userID:     "abcd",
			wantErr:    false,
		},
		{
			name:       "invalid base prefix",
			basePrefix: "invalid",
			randomBit:  0,
			userID:     "abcd",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, err := generateDelegatedPrefix(tt.basePrefix, tt.randomBit, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, prefix)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, prefix)
				assert.Contains(t, prefix, "/64")

				// Verify prefix contains user ID
				assert.Contains(t, prefix, tt.userID)
			}
		})
	}
}

// Test for generateThirdPrefix function
func TestGenerateThirdPrefix(t *testing.T) {
	tests := []struct {
		name       string
		basePrefix string
		userID     string
		wantErr    bool
	}{
		{
			name:       "valid third prefix generation",
			basePrefix: "2a06:1234:5600::/48",
			userID:     "abcd",
			wantErr:    false,
		},
		{
			name:       "invalid base prefix",
			basePrefix: "invalid",
			userID:     "abcd",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, err := generateThirdPrefix(tt.basePrefix, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, prefix)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, prefix)
				assert.Contains(t, prefix, "/64")

				// Verify prefix contains user ID
				assert.Contains(t, prefix, tt.userID)

				// Verify format is correct (no random hex in third tercet)
				assert.Contains(t, prefix, "2a06:1234:5600")
			}
		})
	}
}

// Test for parsePrefix function
func TestParsePrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
	}{
		{
			name:    "valid IPv6 prefix",
			prefix:  "2a05:dfc1:3c00::/44",
			wantErr: false,
		},
		{
			name:    "invalid IPv6 prefix",
			prefix:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipNet, err := parsePrefix(tt.prefix)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ipNet)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ipNet)
			}
		})
	}
}

// Test for validateIPv6Address function
func TestValidateIPv6Address(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{
			name:    "valid IPv6 address",
			address: "2a05:dfc1:3c0F:abcd::1",
			wantErr: false,
		},
		{
			name:    "invalid IPv6 address",
			address: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIPv6Address(tt.address)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
