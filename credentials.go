package vrpc

import (
	"fmt"
	"slices"
	"strings"
)

// EncodeCredentials formats Portal actor credentials as "field value, field value".
// Field names are case-insensitive; ambiguous or unrepresentable entries fail.
// A nil or empty map produces an empty value for anonymous calls. Credential
// schema validation and authentication remain the gateway's responsibility.
func EncodeCredentials(credentials map[string]string) (string, error) {
	names := make([]string, 0, len(credentials))
	seen := map[string]bool{}
	for name, value := range credentials {
		lower := strings.ToLower(name)
		if !credentialName(name) || seen[lower] || value == "" || strings.TrimSpace(value) != value || strings.ContainsAny(value, ",\r\n") {
			return "", fmt.Errorf("vrpc: invalid credential field %q", name)
		}
		for _, c := range value {
			if c < 32 || c == 127 {
				return "", fmt.Errorf("vrpc: invalid credential value")
			}
		}
		seen[lower] = true
		names = append(names, name)
	}
	slices.Sort(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, name+" "+credentials[name])
	}
	return strings.Join(parts, ", "), nil
}
func credentialName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}
