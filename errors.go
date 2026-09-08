package snowflake

type errorKind string

func (e errorKind) Error() string {
	return string(e)
}

// Errors are immutable and may be inspected with errors.Is. They never include
// input contents. JSON decoding through encoding/json may also return its own
// syntax or type errors before calling ID.UnmarshalJSON.
const (
	// ErrSyntax indicates empty input, invalid decimal digits, or an unsupported
	// JSON representation (including negative numbers, fractions, and exponents).
	ErrSyntax = errorKind("snowflake: invalid decimal or JSON syntax")
	// ErrRange indicates a decimal value greater than math.MaxUint64, or a
	// negative SQL integer.
	ErrRange = errorKind("snowflake: identifier outside uint64 range")
	// ErrNilReceiver indicates a nil *ID passed to UnmarshalText, UnmarshalJSON,
	// or Scan, regardless of the input.
	ErrNilReceiver = errorKind("snowflake: nil receiver")
	// ErrSource indicates a SQL source other than string, []byte, int64, or nil.
	ErrSource = errorKind("snowflake: unsupported SQL source type")
	// ErrTimeRange indicates a time outside the representable timestamp interval.
	ErrTimeRange = errorKind("snowflake: time outside Discord timestamp range")
)
