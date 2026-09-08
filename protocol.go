package vrpc

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"uuid"
)

// Protocol header names shared by vRPC clients and gateways.
const (
	HeaderClient        = "vrpc-client"
	HeaderTrace         = "vrpc-trace"
	HeaderOptions       = "vrpc-options"
	HeaderStatus        = "vrpc-status"
	HeaderServer        = "vrpc-server"
	HeaderPortalTraceID = "portal-trace-id"
	ContentTypeJSON     = "application/vrpc+json"
	ContentTypeCBOR     = "application/vrpc+cbor"
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
var pathPattern = regexp.MustCompile(`^[A-Za-z0-9_.]+/[A-Za-z0-9_]+$`)

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
	return "name=" + identity.Name + ",version=" + identity.Version + ",instanceId=" + identity.InstanceID, nil
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

func decodeFields(value string) (map[string]string, error) {
	fields := map[string]string{}
	for part := range strings.SplitSeq(value, ",") {
		key, value, ok := strings.Cut(part, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || key == "" || value == "" || fields[key] != "" {
			return nil, fmt.Errorf("vrpc: malformed metadata header")
		}
		fields[key] = value
	}
	return fields, nil
}

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
	return "id=" + trace.ID + ",span=" + trace.Span, nil
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
