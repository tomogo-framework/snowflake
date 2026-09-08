package snowflake_test

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tomogo-framework/snowflake"
)

var (
	_ fmt.Stringer             = snowflake.ID(0)
	_ encoding.TextMarshaler   = snowflake.ID(0)
	_ encoding.TextAppender    = snowflake.ID(0)
	_ encoding.TextUnmarshaler = (*snowflake.ID)(nil)
	_ json.Marshaler           = snowflake.ID(0)
	_ json.Unmarshaler         = (*snowflake.ID)(nil)
	_ driver.Valuer            = snowflake.ID(0)
	_ sql.Scanner              = (*snowflake.ID)(nil)
)

func TestDecimal(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  snowflake.ID
		err   error
	}{
		{
			name: "empty",
			err:  snowflake.ErrSyntax,
		},
		{
			name:  "zero",
			input: "0",
		},
		{
			name:  "zeros",
			input: "000",
		},
		{
			name:  "leading zeros",
			input: "00042",
			want:  42,
		},
		{
			name:  "long zeros",
			input: strings.Repeat("0", 10000) + "42",
			want:  42,
		},
		{
			name:  "example",
			input: "175928847299117063",
			want:  175928847299117063,
		},
		{
			name:  "above signed",
			input: "9223372036854775808",
			want:  9223372036854775808,
		},
		{
			name:  "max",
			input: "18446744073709551615",
			want:  math.MaxUint64,
		},
		{
			name:  "overflow",
			input: "18446744073709551616",
			err:   snowflake.ErrRange,
		},
		{
			name:  "max padded",
			input: "00018446744073709551615",
			want:  math.MaxUint64,
		},
		{
			name:  "overflow padded",
			input: "00018446744073709551616",
			err:   snowflake.ErrRange,
		},
		{
			name:  "invalid above lexical maximum",
			input: "1844674407370955161x",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "invalid below lexical maximum",
			input: "1000000000000000000x",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "long overflow",
			input: strings.Repeat("9", 10000),
			err:   snowflake.ErrRange,
		},
		{
			name:  "long invalid",
			input: strings.Repeat("9", 10000) + "x",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "plus",
			input: "+1",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "negative",
			input: "-1",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "negative zero",
			input: "-0",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "prefix space",
			input: " 1",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "suffix space",
			input: "1 ",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "newline",
			input: "1\n",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "fraction",
			input: "1.0",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "exponent",
			input: "1e2",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "separator",
			input: "1_000",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "hex",
			input: "0x10",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "letter",
			input: "12x",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "nul",
			input: "1\x00",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "unicode digits",
			input: "１２",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "arabic digit",
			input: "١",
			err:   snowflake.ErrSyntax,
		},
		{
			name:  "invalid utf8",
			input: "1\xff",
			err:   snowflake.ErrSyntax,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, parse := range []func() (snowflake.ID, error){
				func() (snowflake.ID, error) {
					return snowflake.Parse(tc.input)
				},
				func() (snowflake.ID, error) {
					return snowflake.ParseBytes([]byte(tc.input))
				},
			} {
				got, err := parse()
				if got != tc.want || !errors.Is(err, tc.err) {
					t.Fatalf("parse = %v, %v; want %v, %v", got, err, tc.want, tc.err)
				}
			}

			for _, decode := range []func(*snowflake.ID) error{
				func(id *snowflake.ID) error {
					return id.UnmarshalText([]byte(tc.input))
				},
				func(id *snowflake.ID) error {
					return id.Scan(tc.input)
				},
				func(id *snowflake.ID) error {
					return id.Scan([]byte(tc.input))
				},
			} {
				id := snowflake.ID(99)
				err := decode(&id)
				want := tc.want
				if tc.err != nil {
					want = 99
				}

				if id != want || !errors.Is(err, tc.err) {
					t.Fatalf("decode = %v, %v; want %v, %v", id, err, want, tc.err)
				}

				if err != nil && len(err.Error()) > 100 {
					t.Fatal("error includes excessive input")
				}
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	if snowflake.MustParse("42") != 42 {
		t.Fatal("valid constant")
	}

	defer func() {
		err, ok := recover().(error)
		if !ok || !errors.Is(err, snowflake.ErrSyntax) {
			t.Fatalf("panic = %v", err)
		}
	}()
	snowflake.MustParse("invalid")
}

func TestParts(t *testing.T) {
	for _, tc := range []struct {
		id        snowflake.ID
		millis    int64
		worker    uint8
		process   uint8
		increment uint16
	}{
		{
			id:     0,
			millis: 1420070400000,
		},
		{
			id:        175928847299117063,
			millis:    1462015105796,
			worker:    1,
			increment: 7,
		},
		{
			id:     131072,
			millis: 1420070400000,
			worker: 1,
		},
		{
			id:      4096,
			millis:  1420070400000,
			process: 1,
		},
		{
			id:        4095,
			millis:    1420070400000,
			increment: 4095,
		},
		{
			id:     4194304,
			millis: 1420070400001,
		},
		{
			id:        math.MaxUint64,
			millis:    5818116911103,
			worker:    31,
			process:   31,
			increment: 4095,
		},
	} {
		got := tc.id.Time()
		if got.UnixMilli() != tc.millis || got.Location() != time.UTC || got.Nanosecond()%1000000 != 0 {
			t.Errorf("%v Time = %v", tc.id, got)
		}

		if tc.id.WorkerID() != tc.worker || tc.id.ProcessID() != tc.process || tc.id.Increment() != tc.increment {
			t.Errorf("%v parts = %d, %d, %d", tc.id, tc.id.WorkerID(), tc.id.ProcessID(), tc.id.Increment())
		}
	}
}

func TestFormattingAndOwnership(t *testing.T) {
	for _, id := range []snowflake.ID{0, 1, 99, 100, 175928847299117063, 9223372036854775808, math.MaxUint64} {
		want := strconv.FormatUint(uint64(id), 10)
		text, textErr := id.MarshalText()
		data, jsonErr := id.MarshalJSON()
		if textErr != nil || jsonErr != nil || id.String() != want || string(text) != want || string(data) != `"`+want+`"` {
			t.Fatalf("formatting %v: %s, %s, %v, %v", id, text, data, textErr, jsonErr)
		}

		for _, capacity := range []int{3, 32} {
			buf := make([]byte, 3, capacity)
			copy(buf, "id=")
			got, err := id.AppendText(buf)
			if err != nil || string(got) != "id="+want || string(buf) != "id=" {
				t.Fatalf("AppendText = %s, %v", got, err)
			}

			if capacity == 32 && &got[0] != &buf[0] {
				t.Fatal("did not reuse sufficient capacity")
			}
		}

		got, err := id.AppendText(nil)
		if err != nil || string(got) != want {
			t.Fatal("nil destination append")
		}

		text[0] = 'x'
		data[1] = 'x'
		freshText, _ := id.MarshalText()
		freshJSON, _ := id.MarshalJSON()
		if string(freshText) != want || string(freshJSON) != `"`+want+`"` {
			t.Fatal("marshal buffers are shared")
		}
	}

	input := []byte("42")
	var id snowflake.ID
	if err := id.Scan(input); err != nil || string(input) != "42" {
		t.Fatal("Scan mutated input or failed")
	}

	input[0] = '9'
	if id != 42 {
		t.Fatal("Scan retained input")
	}
}

func TestJSON(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  snowflake.ID
		err   error
	}{
		{
			input: `"175928847299117063"`,
			want:  175928847299117063,
		},
		{
			input: `175928847299117063`,
			want:  175928847299117063,
		},
		{
			input: `9007199254740993`,
			want:  9007199254740993,
		},
		{
			input: `18446744073709551615`,
			want:  math.MaxUint64,
		},
		{
			input: `"18446744073709551615"`,
			want:  math.MaxUint64,
		},
		{
			input: `0`,
		},
		{
			input: `"000"`,
		},
		{
			input: `"00042"`,
			want:  42,
		},
		{
			input: `"\u0034\u0032"`,
			want:  42,
		},
		{
			input: `"4\u0032"`,
			want:  42,
		},
		{
			input: `null`,
		},
		{
			input: " \r\n\tnull \r\n\t",
		},
		{
			input: " \r\n\t42 \r\n\t",
			want:  42,
		},
		{
			input: " \r\n\t\"42\" \r\n\t",
			want:  42,
		},
		{
			input: `18446744073709551616`,
			err:   snowflake.ErrRange,
		},
		{
			input: `"18446744073709551616"`,
			err:   snowflake.ErrRange,
		},
		{
			input: ``,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `""`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"42`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `42"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"42"0`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"4""2"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"\u0034""2"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `01`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `00`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `+1`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `-0`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `-1`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `1.0`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `1.`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `.1`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `1e0`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `1E+2`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `1e`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `0x1`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `NaN`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `Infinity`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `true`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `{}`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `[]`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `null null`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `42 43`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `" 42"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"+42"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"\t42"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"\x34"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"\u003"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"\uD800"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"\uD800\uDC00"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"\"42"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"\\42"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"\/42"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: `"１２"`,
			err:   snowflake.ErrSyntax,
		},
		{
			input: "\"4\xff2\"",
			err:   snowflake.ErrSyntax,
		},
		{
			input: "\"4\x002\"",
			err:   snowflake.ErrSyntax,
		},
		{
			input: "\v42",
			err:   snowflake.ErrSyntax,
		},
		{
			input: "\u00a042",
			err:   snowflake.ErrSyntax,
		},
	} {
		t.Run(fmt.Sprintf("%q", tc.input), func(t *testing.T) {
			for _, direct := range []bool{true, false} {
				id := snowflake.ID(99)
				var err error
				if direct {
					err = id.UnmarshalJSON([]byte(tc.input))
				} else {
					err = json.Unmarshal([]byte(tc.input), &id)
				}

				want := tc.want
				if tc.err != nil {
					want = 99
				}

				if id != want || (err == nil) != (tc.err == nil) {
					t.Fatalf("direct=%v: %v, %v; want %v, %v", direct, id, err, want, tc.err)
				}

				if direct && !errors.Is(err, tc.err) {
					t.Fatalf("error kind: %v, want %v", err, tc.err)
				}
			}
		})
	}
}

func TestJSONDecimalEscapes(t *testing.T) {
	for _, digits := range []string{"0123456789", "00042", "175928847299117063", "18446744073709551615"} {
		want, err := strconv.ParseUint(digits, 10, 64)
		if err != nil {
			t.Fatal(err)
		}

		for _, mixed := range []bool{false, true} {
			var data strings.Builder
			data.WriteByte('"')
			for i := range len(digits) {
				if mixed && i%2 == 0 {
					data.WriteByte(digits[i])
				} else {
					fmt.Fprintf(&data, `\u%04x`, digits[i])
				}
			}
			data.WriteByte('"')
			var id snowflake.ID
			if err := id.UnmarshalJSON([]byte(data.String())); err != nil || uint64(id) != want {
				t.Fatalf("decode(%s) = %v, %v; want %d", data.String(), id, err, want)
			}
		}
	}

	// Go string escapes must not bypass JSON's stricter grammar.
	for _, input := range []string{`"\064\062"`, `"\U00000034"`, `"\x34"`, `"\a"`, `"\v"`} {
		id := snowflake.ID(99)
		if err := id.UnmarshalJSON([]byte(input)); !errors.Is(err, snowflake.ErrSyntax) || id != 99 {
			t.Fatalf("decode(%s) = %v, %v", input, id, err)
		}
	}
}

func TestSQL(t *testing.T) {
	for _, tc := range []struct {
		src  any
		want snowflake.ID
		err  error
	}{
		{
			src: nil,
		},
		{
			src: int64(0),
		},
		{
			src:  int64(math.MaxInt64),
			want: math.MaxInt64,
		},
		{
			src: int64(-1),
			err: snowflake.ErrRange,
		},
		{
			src: int64(math.MinInt64),
			err: snowflake.ErrRange,
		},
		{
			src: []byte(nil),
			err: snowflake.ErrSyntax,
		},
		{
			src:  "18446744073709551615",
			want: math.MaxUint64,
		},
		{
			src:  []byte("18446744073709551615"),
			want: math.MaxUint64,
		},
		{
			src: uint64(1),
			err: snowflake.ErrSource,
		},
		{
			src: int(1),
			err: snowflake.ErrSource,
		},
		{
			src: float64(1),
			err: snowflake.ErrSource,
		},
		{
			src: true,
			err: snowflake.ErrSource,
		},
		{
			src: json.Number("1"),
			err: snowflake.ErrSource,
		},
		{
			src: (*string)(nil),
			err: snowflake.ErrSource,
		},
	} {
		id := snowflake.ID(99)
		err := id.Scan(tc.src)
		want := tc.want
		if tc.err != nil {
			want = 99
		}

		if id != want || !errors.Is(err, tc.err) {
			t.Errorf("Scan(%T) = %v, %v; want %v, %v", tc.src, id, err, want, tc.err)
		}
	}

	for _, id := range []snowflake.ID{0, 175928847299117063, math.MaxUint64} {
		value, err := driver.DefaultParameterConverter.ConvertValue(id)
		if err != nil || value != strconv.FormatUint(uint64(id), 10) {
			t.Errorf("driver value = %#v, %v", value, err)
		}
	}
}

func TestNilReceivers(t *testing.T) {
	var id *snowflake.ID
	for _, data := range [][]byte{nil, []byte("42"), []byte(`"42"`), []byte("null"), []byte("invalid")} {
		if !errors.Is(id.UnmarshalText(data), snowflake.ErrNilReceiver) || !errors.Is(id.UnmarshalJSON(data), snowflake.ErrNilReceiver) {
			t.Fatal("nil decoder receiver")
		}
	}

	for _, src := range []any{nil, int64(-1), int64(42), "42", []byte("42"), true} {
		if !errors.Is(id.Scan(src), snowflake.ErrNilReceiver) {
			t.Fatal("nil scanner receiver")
		}
	}

	data, err := json.Marshal(id)
	if err != nil || string(data) != "null" {
		t.Fatalf("standard nil pointer encoding = %s, %v", data, err)
	}
}

func TestStructsAndMapKeys(t *testing.T) {
	type record struct {
		ID     snowflake.ID            `json:"id"`
		Absent snowflake.ID            `json:"absent,omitempty"`
		IDs    []snowflake.ID          `json:"ids"`
		ByID   map[snowflake.ID]string `json:"by_id"`
	}
	want := record{
		ID:  math.MaxUint64,
		IDs: []snowflake.ID{0, 9007199254740993},
		ByID: map[snowflake.ID]string{
			175928847299117063: "example",
		},
	}
	data, err := json.Marshal(want)
	if err != nil || string(data) != `{"id":"18446744073709551615","ids":["0","9007199254740993"],"by_id":{"175928847299117063":"example"}}` {
		t.Fatalf("struct encoding = %s, %v", data, err)
	}

	var got record
	if err := json.Unmarshal(data, &got); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("struct decoding = %+v, %v", got, err)
	}

	var keys map[snowflake.ID]string
	if err := json.Unmarshal([]byte(`{"\u0034\u0032":"escaped","0001":"leading"}`), &keys); err != nil || keys[42] != "escaped" || keys[1] != "leading" {
		t.Fatalf("map decoding = %v, %v", keys, err)
	}

	if err := json.Unmarshal([]byte(`{"-1":"invalid"}`), &keys); err == nil {
		t.Fatal("invalid map key accepted")
	}
}

func TestIndependentGoroutines(t *testing.T) {
	var wg sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wg.Go(func() {
			for i := 0; i < 100; i++ {
				id := snowflake.ID(175928847299117063)
				buf, _ := id.AppendText(make([]byte, 0, 32))
				var decoded snowflake.ID
				if err := decoded.Scan(buf); err != nil || decoded != id {
					t.Errorf("concurrent scan = %v, %v", decoded, err)
				}

				data, _ := json.Marshal(id)
				if err := json.Unmarshal(data, &decoded); err != nil || decoded != id {
					t.Errorf("concurrent JSON = %v, %v", decoded, err)
				}

				if !bytes.Equal(buf, []byte("175928847299117063")) || id.Time().UnixMilli() != 1462015105796 {
					t.Error("concurrent extraction or formatting")
				}
			}
		})
	}
	wg.Wait()
}
