package skel

import (
	"time"
	"uuid"

	"cloud.google.com/go/civil"
	"github.com/shopspring/decimal"
	internalskel "go.yorun.ai/vrpc/internal/skel"
)

// Sensitive is implemented by generated values that are sensitive as a whole.
type Sensitive = internalskel.Sensitive

// Decimal is the runtime representation of the Skel decimal scalar.
type Decimal = internalskel.Decimal

// Binary is the runtime representation of the Skel binary scalar.
type Binary = internalskel.Binary

// UUID is the runtime representation of the Skel uuid scalar.
type UUID = internalskel.UUID

// JSON is the runtime representation of arbitrary Skel JSON data.
type JSON = internalskel.JSON

// Timestamp is the runtime representation of an absolute Skel timestamp.
type Timestamp = internalskel.Timestamp

// Duration is the runtime representation of a Skel duration.
type Duration = internalskel.Duration

// LocalDate is the runtime representation of a date without a time zone.
type LocalDate = internalskel.LocalDate

// LocalTime is the runtime representation of a time of day without a time zone.
type LocalTime = internalskel.LocalTime

// LocalDateTime is the runtime representation of a local date and time without a time zone.
type LocalDateTime = internalskel.LocalDateTime

// NewDecimal converts value to the Skel decimal representation.
func NewDecimal(value decimal.Decimal) Decimal {
	return internalskel.NewDecimal(value)
}

// NewTimestamp converts t to the Skel timestamp representation.
func NewTimestamp(t time.Time) Timestamp {
	return internalskel.NewTimestamp(t)
}

// NewDuration converts d to the Skel duration representation.
func NewDuration(d time.Duration) Duration {
	return internalskel.NewDuration(d)
}

// NewUUID converts id to the Skel UUID representation.
func NewUUID(id uuid.UUID) UUID {
	return internalskel.NewUUID(id)
}

// NewLocalDate converts date to the Skel local-date representation.
func NewLocalDate(date civil.Date) LocalDate {
	return internalskel.NewLocalDate(date)
}

// NewLocalDateOf extracts the local date from t.
func NewLocalDateOf(t time.Time) LocalDate {
	return internalskel.NewLocalDateOf(t)
}

// NewLocalTime converts clock to the Skel local-time representation.
func NewLocalTime(clock civil.Time) LocalTime {
	return internalskel.NewLocalTime(clock)
}

// NewLocalTimeOf extracts the local time from t.
func NewLocalTimeOf(t time.Time) LocalTime {
	return internalskel.NewLocalTimeOf(t)
}

// NewLocalDateTime converts dateTime to the Skel local-date-time representation.
func NewLocalDateTime(dateTime civil.DateTime) LocalDateTime {
	return internalskel.NewLocalDateTime(dateTime)
}

// NewLocalDateTimeOf extracts the local date and time from t.
func NewLocalDateTimeOf(t time.Time) LocalDateTime {
	return internalskel.NewLocalDateTimeOf(t)
}

// NewTimestampNow returns the current UTC timestamp.
func NewTimestampNow() Timestamp {
	return internalskel.NewTimestampNow()
}
