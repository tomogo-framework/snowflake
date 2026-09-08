// Package snowflake implements Discord's 64-bit, time-sortable identifiers.
// It parses and serializes IDs without floating-point conversion, and provides
// timestamp boundaries for pagination. It does not issue Discord identifiers.
package snowflake

import (
	"strconv"
	"time"
)

// ID is a Discord snowflake backed by uint64. It is comparable and usable as a
// map key. Zero represents an absent identifier by convention; it still formats
// as "0" and its Time is Discord's epoch. All uint64 values are representable;
// parsing does not establish that Discord issued an ID or that it exists.
type ID uint64

func parseDecimal[S ~string | ~[]byte](s S) (ID, error) {
	if len(s) == 0 {
		return 0, ErrSyntax
	}

	// Bound strconv's error allocations without limiting valid leading zeros.
	for len(s) > 1 && s[0] == '0' {
		s = s[1:]
	}

	// Equal-width decimal strings sort numerically. Reject overflow before
	// strconv constructs an error, but still check every byte for invalid digits.
	if len(s) >= 20 {
		if len(s) > 20 || string(s) > "18446744073709551615" {
			for i := 0; i < len(s); i++ {
				if s[i] < '0' || s[i] > '9' {
					return 0, ErrSyntax
				}
			}

			return 0, ErrRange
		}
	}

	v, err := strconv.ParseUint(string(s), 10, 64)
	if err != nil {
		return 0, ErrSyntax
	}

	return ID(v), nil
}

// Epoch is Discord's epoch, 2015-01-01T00:00:00Z, in Unix milliseconds.
const Epoch int64 = 1420070400000

// Parse parses one or more ASCII decimal digits in the full uint64 range.
// Zero and leading zeros are accepted. Signs, whitespace, non-ASCII digits,
// invalid UTF-8, and empty input are rejected. Errors are ErrSyntax or ErrRange;
// the returned ID is zero on failure.
func Parse(s string) (ID, error) {
	return parseDecimal(s)
}

// ParseBytes is Parse for a byte slice. It does not retain or modify data.
func ParseBytes(data []byte) (ID, error) {
	return parseDecimal(data)
}

// MustParse is Parse but panics with its error on invalid input. It is intended
// for known constants, such as test fixtures.
func MustParse(s string) ID {
	id, err := Parse(s)
	if err != nil {
		panic(err)
	}

	return id
}

// String returns decimal digits without leading zeros, except for zero itself.
func (id ID) String() string {
	return strconv.FormatUint(uint64(id), 10)
}

// Time returns the encoded timestamp at millisecond precision in UTC. Even
// zero has an encoded time (Epoch); it does not return time.Time's zero value.
func (id ID) Time() time.Time {
	millis := int64(uint64(id)>>22) + Epoch

	return time.UnixMilli(millis).UTC()
}

// WorkerID returns the five-bit internal worker identifier (0 through 31).
func (id ID) WorkerID() uint8 {
	return uint8((uint64(id) & 0x3E0000) >> 17)
}

// ProcessID returns the five-bit internal process identifier (0 through 31).
func (id ID) ProcessID() uint8 {
	return uint8((uint64(id) & 0x1F000) >> 12)
}

// Increment returns the twelve-bit increment (0 through 4095).
func (id ID) Increment() uint16 {
	return uint16(uint64(id) & 0xFFF)
}
