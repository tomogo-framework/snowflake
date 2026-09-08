package snowflake

import "database/sql/driver"

// Value implements driver.Valuer with a decimal string, including "0" for zero.
// Use a database column that can hold the full unsigned range, such as text or
// DECIMAL(20,0); a signed BIGINT cannot hold all IDs. It never returns an error.
func (id ID) Value() (driver.Value, error) {
	return id.String(), nil
}

// Scan implements sql.Scanner. It accepts string and []byte using Parse's policy,
// nonnegative int64 directly, and nil (resetting the receiver to zero). A typed
// nil []byte is empty input, not SQL NULL. Byte slices are neither retained nor
// modified. Errors leave the receiver unchanged; a nil receiver returns
// ErrNilReceiver. Unsupported types return ErrSource, negative int64 values
// return ErrRange, and decimal input returns Parse's errors.
func (id *ID) Scan(src any) error {
	if id == nil {
		return ErrNilReceiver
	}

	var (
		v   ID
		err error
	)
	switch src := src.(type) {
	case string:
		v, err = Parse(src)
	case []byte:
		v, err = ParseBytes(src)
	case int64:
		if src < 0 {
			return ErrRange
		}

		v = ID(src)
	case nil:
		v = 0
	default:
		return ErrSource
	}

	if err != nil {
		return err
	}

	*id = v

	return nil
}
