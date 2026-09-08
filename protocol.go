package vrpc

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	wire "go.yorun.ai/vrpc/transport/http"
	"regexp"
	"strings"
	"uuid"
)

// Protocol header names shared by vRPC clients and gateways.
const (
	HeaderClient        = wire.HeaderRpcClient
	HeaderTrace         = wire.HeaderRpcTrace
	HeaderOptions       = wire.HeaderRpcOptions
	HeaderStatus        = wire.HeaderRpcStatus
	HeaderServer        = wire.HeaderRpcServer
	HeaderPortalTraceID = "portal-trace-id"
	ContentTypeJSON     = wire.ContentTypeJson
	ContentTypeCBOR     = wire.ContentTypeCbor
	StatusOK            = "OK"
)

// Identity describes a client or server instance, independently of Vine metadata.
type Identity struct {
	Name       string
	Version    string
	InstanceID string
}

var namePattern = regexp.MustCompile(`^[a-z]+(?:\.[a-z]+)*$`)
var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
var statusPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// EncodeIdentity validates and encodes a vRPC instance identity.
func EncodeIdentity(identity Identity) (string, error) {
	if !namePattern.MatchString(identity.Name) {
		return "", fmt.Errorf("vrpc: invalid identity name")
	}
	parts := versionPattern.FindStringSubmatch(identity.Version)
	if parts == nil {
		return "", fmt.Errorf("vrpc: invalid semantic version")
	}
	for _, part := range strings.Split(parts[4], ".") {
		if len(part) > 1 && part[0] == '0' && strings.Trim(part, "0123456789") == "" {
			return "", fmt.Errorf("vrpc: invalid prerelease version")
		}
	}
	if _, err := uuid.Parse(identity.InstanceID); err != nil {
		return "", fmt.Errorf("vrpc: invalid instance UUID: %w", err)
	}
	return wire.EncodeApp(identity.Name, identity.Version, identity.InstanceID), nil
}

// DecodeIdentity validates a vrpc-client or vrpc-server header.
func DecodeIdentity(value string) (Identity, error) {
	fields, err := decodeFields(value)
	if err != nil {
		return Identity{}, err
	}
	identity := Identity{Name: fields["name"], Version: fields["version"], InstanceID: fields["instanceId"]}
	_, err = EncodeIdentity(identity)
	return identity, err
}

func decodeFields(value string) (map[string]string, error) { return wire.DecodeFields(value) }

// Trace contains a vRPC trace ID and span ID as lowercase hexadecimal strings.
type Trace struct {
	ID   string
	Span string
}

// NewTrace creates a new trace and span.
func NewTrace() Trace { return Trace{ID: randomHex(16), Span: randomHex(8)} }

// Child preserves the trace ID and creates a new span for an outgoing call.
func (t Trace) Child() Trace { return Trace{ID: t.ID, Span: randomHex(8)} }

// EncodeTrace validates and encodes a trace. A span is required.
func EncodeTrace(trace Trace) (string, error) {
	if !validHex(trace.ID, 32) || !validHex(trace.Span, 16) {
		return "", fmt.Errorf("vrpc: invalid trace or span ID")
	}
	return wire.EncodeTrace(trace.ID, trace.Span), nil
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
