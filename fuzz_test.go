package snowflake_test

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/tomogo-framework/snowflake"
)

func referenceDecimal(s string) (uint64, bool) {
	if len(s) == 0 {
		return 0, false
	}

	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}

	v, err := strconv.ParseUint(s, 10, 64)

	return v, err == nil
}

func referenceJSON(data []byte) (uint64, bool) {
	if !json.Valid(data) {
		return 0, false
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil {
		return 0, false
	}

	switch value := value.(type) {
	case nil:
		return 0, true
	case string:
		return referenceDecimal(value)
	case json.Number:
		return referenceDecimal(string(value))
	default:
		return 0, false
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"", "0", "0001", "175928847299117063", "18446744073709551615", "18446744073709551616", "+1", "-0", "1\xff", "１２"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		want, valid := referenceDecimal(s)
		got, err := snowflake.Parse(s)
		fromBytes, bytesErr := snowflake.ParseBytes([]byte(s))
		if (err == nil) != valid || (bytesErr == nil) != valid || (valid && uint64(got) != want) || got != fromBytes {
			t.Fatalf("parse mismatch: %v, %v; bytes %v, %v; want %d, valid %v", got, err, fromBytes, bytesErr, want, valid)
		}

		id := snowflake.ID(99)
		decodeErr := id.UnmarshalText([]byte(s))
		if (decodeErr == nil) != valid || (!valid && id != 99) || (valid && uint64(id) != want) {
			t.Fatal("text decode mismatch or partial modification")
		}
	})
}

func FuzzUnmarshalJSON(f *testing.F) {
	for _, s := range []string{`"175928847299117063"`, `18446744073709551615`, `"\u0034\u0032"`, " \nnull\t", `01`, `-0`, `1e0`, `"\uD800"`, "\"4\xff2\"", `"42""3"`, `""`, `18446744073709551616`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		want, valid := referenceJSON(data)
		id := snowflake.ID(99)
		err := id.UnmarshalJSON(data)
		if (err == nil) != valid || (!valid && id != 99) || (valid && uint64(id) != want) {
			t.Fatalf("direct JSON = %v, %v; want %d, valid %v", id, err, want, valid)
		}

		id = 99
		err = json.Unmarshal(data, &id)
		if (err == nil) != valid || (!valid && id != 99) || (valid && uint64(id) != want) {
			t.Fatalf("standard JSON = %v, %v; want %d, valid %v", id, err, want, valid)
		}
	})
}

func FuzzID(f *testing.F) {
	for _, value := range []uint64{0, 1, 175928847299117063, 9223372036854775808, math.MaxUint64} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value uint64) {
		id := snowflake.ID(value)
		want := strconv.FormatUint(value, 10)
		buf, _ := id.AppendText([]byte("prefix:"))
		if id.String() != want || string(buf) != "prefix:"+want {
			t.Fatal("format mismatch")
		}

		encoded, _ := json.Marshal(id)
		var decoded snowflake.ID
		if string(encoded) != `"`+want+`"` || json.Unmarshal(encoded, &decoded) != nil || decoded != id {
			t.Fatal("JSON precision or round trip")
		}

		if id.Time().UnixMilli() != int64(value/4194304)+1420070400000 || uint64(id.WorkerID()) != value/131072%32 || uint64(id.ProcessID()) != value/4096%32 || uint64(id.Increment()) != value%4096 {
			t.Fatal("bit extraction mismatch")
		}

		stored, _ := id.Value()
		if decoded.Scan(stored) != nil || decoded != id {
			t.Fatal("SQL round trip")
		}
	})
}

func FuzzTimeBounds(f *testing.F) {
	for _, millis := range []int64{math.MinInt64, 1420070399999, 1420070400000, 1462015105796, 5818116911103, 5818116911104, math.MaxInt64} {
		f.Add(millis)
	}
	f.Fuzz(func(t *testing.T, millis int64) {
		instant := time.UnixMilli(millis)
		min, minErr := snowflake.MinID(instant)
		max, maxErr := snowflake.MaxID(instant)
		valid := millis >= 1420070400000 && millis <= 5818116911103
		if (minErr == nil) != valid || (maxErr == nil) != valid {
			t.Fatal("timestamp range mismatch")
		}

		if valid {
			want := uint64(millis-1420070400000) * 4194304
			if uint64(min) != want || uint64(max) != want+4194303 {
				t.Fatal("timestamp boundary mismatch")
			}
		} else if min != 0 || max != 0 {
			t.Fatal("nonzero result on error")
		}
	})
}
