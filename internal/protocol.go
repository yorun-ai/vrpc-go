package vrpc

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	rpchttp "go.yorun.ai/vrpc/transport/http"
	"regexp"
	"strings"
	"uuid"
)

type Identity struct {
	Name       string
	Version    string
	InstanceID string
}

var namePattern = regexp.MustCompile(`^[a-z]+(?:\.[a-z]+)*$`)
var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
var statusPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func EncodeIdentity(identity Identity) (string, error) {
	if !namePattern.MatchString(identity.Name) {
		return "", fmt.Errorf("vrpc: invalid identity name")
	}
	parts := versionPattern.FindStringSubmatch(identity.Version)
	if parts == nil {
		return "", fmt.Errorf("vrpc: invalid semantic version")
	}
	for part := range strings.SplitSeq(parts[4], ".") {
		if len(part) > 1 && part[0] == '0' && strings.Trim(part, "0123456789") == "" {
			return "", fmt.Errorf("vrpc: invalid prerelease version")
		}
	}
	if _, err := uuid.Parse(identity.InstanceID); err != nil {
		return "", fmt.Errorf("vrpc: invalid instance UUID: %w", err)
	}
	return rpchttp.EncodeApp(identity.Name, identity.Version, identity.InstanceID), nil
}

func DecodeIdentity(value string) (Identity, error) {
	fields, err := decodeFields(value)
	if err != nil {
		return Identity{}, err
	}
	identity := Identity{
		Name:       fields["name"],
		Version:    fields["version"],
		InstanceID: fields["instanceId"],
	}
	_, err = EncodeIdentity(identity)
	return identity, err
}

func decodeFields(value string) (map[string]string, error) {
	return rpchttp.DecodeFields(value)
}

type Trace struct {
	ID   string
	Span string
}

func NewTrace() Trace {
	return Trace{
		ID:   randomHex(16),
		Span: randomHex(8),
	}
}

func (t Trace) Child() Trace {
	return Trace{
		ID:   t.ID,
		Span: randomHex(8),
	}
}

func EncodeTrace(trace Trace) (string, error) {
	if !validHex(trace.ID, 32) || !validHex(trace.Span, 16) {
		return "", fmt.Errorf("vrpc: invalid trace or span ID")
	}
	return rpchttp.EncodeTrace(trace.ID, trace.Span), nil
}

func validHex(value string, length int) bool {
	if len(value) != length || strings.ToLower(value) != value || strings.Trim(value, "0") == "" {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func randomHex(size int) string {
	for {
		b := make([]byte, size)
		_, _ = rand.Read(b)
		value := hex.EncodeToString(b)
		if strings.Trim(value, "0") != "" {
			return value
		}
	}
}
