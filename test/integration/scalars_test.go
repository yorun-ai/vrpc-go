package integration_test

import (
	"bytes"
	"encoding/json/v2"
	"testing"
	"time"
	"uuid"

	"github.com/fxamacker/cbor/v2"
	"github.com/shopspring/decimal"
	vineskel "go.yorun.ai/vine/core/skel"
	clientskel "go.yorun.ai/vrpc/skel"
)

func TestScalarWireCompatibility(t *testing.T) {
	instant := time.Date(2026, 9, 9, 12, 34, 56, 123456789, time.FixedZone("local", 8*3600))
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	cases := []struct {
		name      string
		client    any
		server    any
		newClient func() any
		newServer func() any
	}{
		{
			"decimal",
			clientskel.NewDecimal(decimal.RequireFromString("1.00")),
			vineskel.NewDecimal(decimal.RequireFromString("1.00")),
			func() any {
				return new(clientskel.Decimal)
			},
			func() any {
				return new(vineskel.Decimal)
			},
		},

		{
			"binary",
			clientskel.Binary{
				0,
				255,
			},
			vineskel.Binary{
				0,
				255,
			},
			func() any {
				return new(clientskel.Binary)
			},
			func() any {
				return new(vineskel.Binary)
			},
		},

		{
			"timestamp",
			clientskel.NewTimestamp(instant),
			vineskel.NewTimestamp(instant),
			func() any {
				return new(clientskel.Timestamp)
			},
			func() any {
				return new(vineskel.Timestamp)
			},
		},

		{
			"duration",
			clientskel.NewDuration(-123 * time.Millisecond),
			vineskel.NewDuration(-123 * time.Millisecond),
			func() any {
				return new(clientskel.Duration)
			},
			func() any {
				return new(vineskel.Duration)
			},
		},

		{
			"localDate",
			clientskel.NewLocalDateOf(instant),
			vineskel.NewLocalDateOf(instant),
			func() any {
				return new(clientskel.LocalDate)
			},
			func() any {
				return new(vineskel.LocalDate)
			},
		},

		{
			"localTime",
			clientskel.NewLocalTimeOf(instant),
			vineskel.NewLocalTimeOf(instant),
			func() any {
				return new(clientskel.LocalTime)
			},
			func() any {
				return new(vineskel.LocalTime)
			},
		},

		{
			"localDateTime",
			clientskel.NewLocalDateTimeOf(instant),
			vineskel.NewLocalDateTimeOf(instant),
			func() any {
				return new(clientskel.LocalDateTime)
			},
			func() any {
				return new(vineskel.LocalDateTime)
			},
		},

		{
			"uuid",
			clientskel.NewUUID(id),
			vineskel.NewUUID(id),
			func() any {
				return new(clientskel.UUID)
			},
			func() any {
				return new(vineskel.UUID)
			},
		},

		{
			"json",
			clientskel.JSON(`{"value":42}`),
			vineskel.JSON(`{"value":42}`),
			func() any {
				return new(clientskel.JSON)
			},
			func() any {
				return new(vineskel.JSON)
			},
		},
	}
	codecs := []struct {
		name   string
		encode func(any) ([]byte, error)
		decode func([]byte, any) error
	}{
		{
			"json",
			func(v any) ([]byte, error) {
				return json.Marshal(v)
			},
			func(b []byte, v any) error {
				return json.Unmarshal(b, v)
			},
		},

		{
			"cbor",
			cbor.Marshal,
			cbor.Unmarshal,
		},
	}
	for _, tc := range cases {
		for _, codec := range codecs {
			t.Run(tc.name+"/"+codec.name, func(t *testing.T) {
				clientWire, err := codec.encode(tc.client)
				if err != nil {
					t.Fatal(err)
				}
				serverWire, err := codec.encode(tc.server)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(clientWire, serverWire) {
					t.Fatalf("wire mismatch: client=%x server=%x", clientWire, serverWire)
				}
				for _, target := range []any{
					tc.newClient(),
					tc.newServer(),
				} {
					if err := codec.decode(clientWire, target); err != nil {
						t.Fatal(err)
					}
					roundTrip, err := codec.encode(target)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(clientWire, roundTrip) {
						t.Fatalf("round trip changed wire: %x -> %x", clientWire, roundTrip)
					}
				}
			})
		}
	}
}
