package snowflake

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// AppendText implements encoding.TextAppender, appending decimal digits to dst
// and preserving its existing prefix. It never returns an error. It reuses dst's
// capacity when possible; the caller owns the returned buffer.
func (id ID) AppendText(dst []byte) ([]byte, error) {
	return strconv.AppendUint(dst, uint64(id), 10), nil
}

// MarshalText implements encoding.TextMarshaler, returning decimal digits in a
// new buffer. It never returns an error.
func (id ID) MarshalText() ([]byte, error) {
	// strconv has cached strings for small integers; avoid reserving 20 bytes
	// for the common absent identifier while keeping one allocation for all IDs.
	if id < 100 {
		return []byte(id.String()), nil
	}

	return id.AppendText(make([]byte, 0, 20))
}

// UnmarshalText implements encoding.TextUnmarshaler using ParseBytes's policy.
// On failure the receiver is unchanged. A nil receiver returns ErrNilReceiver.
func (id *ID) UnmarshalText(data []byte) error {
	if id == nil {
		return ErrNilReceiver
	}

	v, err := ParseBytes(data)
	if err != nil {
		return err
	}

	*id = v

	return nil
}

// MarshalJSON implements json.Marshaler, returning a quoted decimal string in a
// new buffer. It never returns an error and preserves the full uint64 range.
func (id ID) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 1, 22)
	buf[0] = '"'
	buf = strconv.AppendUint(buf, uint64(id), 10)

	return append(buf, '"'), nil
}

// UnmarshalJSON accepts a quoted decimal string, a nonnegative JSON integer, or
// null (which resets the receiver to zero). Standard JSON whitespace around the
// value and escapes within strings are supported. Decoded strings follow Parse's
// policy; bare numbers cannot have leading zeros, signs, fractions, or exponents.
// This method validates syntax even when called directly. On any failure the
// receiver is unchanged. A nil receiver returns ErrNilReceiver.
func (id *ID) UnmarshalJSON(data []byte) error {
	if id == nil {
		return ErrNilReceiver
	}

	if len(data) > 0 && (data[0] <= ' ' || data[len(data)-1] <= ' ') {
		data = bytes.Trim(data, " \t\r\n")
	}

	if string(data) == "null" {
		*id = 0

		return nil
	}

	if len(data) == 0 {
		return ErrSyntax
	}

	var (
		v   ID
		err error
	)
	if data[0] == '"' {
		if len(data) < 2 || data[len(data)-1] != '"' {
			return ErrSyntax
		}

		inner := data[1 : len(data)-1]
		if bytes.IndexByte(inner, '\\') < 0 {
			// Decimal bytes cannot contain JSON control characters or delimiters.
			v, err = ParseBytes(inner)
		} else {
			var s string
			if json.Unmarshal(data, &s) != nil {
				return ErrSyntax
			}

			v, err = Parse(s)
		}
	} else {
		if data[0] < '0' || data[0] > '9' || (data[0] == '0' && len(data) > 1) {
			return ErrSyntax
		}

		v, err = ParseBytes(data)
	}

	if err != nil {
		return err
	}

	*id = v

	return nil
}
