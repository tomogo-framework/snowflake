package snowflake

import (
	"encoding/json"
	"testing"
	"time"
)

type benchmarkObject struct {
	ID        ID   `json:"id"`
	ChannelID ID   `json:"channel_id"`
	IDs       []ID `json:"ids"`
}

var (
	benchmarkID     ID
	benchmarkBytes  []byte
	benchmarkString string
	benchmarkError  error
	benchmarkValue  any
	benchmarkTime   time.Time
	benchmarkParts  uint64
)

func BenchmarkParse(b *testing.B) {
	for _, tc := range []struct {
		name string
		text string
	}{
		{
			name: "zero",
			text: "0",
		},
		{
			name: "18digits",
			text: "175928847299117063",
		},
		{
			name: "19digits",
			text: "1412345678901234567",
		},
		{
			name: "max",
			text: "18446744073709551615",
		},
		{
			name: "leadingZeros",
			text: "000175928847299117063",
		},
		{
			name: "empty",
			text: "",
		},
		{
			name: "overflow",
			text: "18446744073709551616",
		},
		{
			name: "invalid",
			text: "17592884729911706x",
		},
	} {
		b.Run(tc.name+"/string", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkID, benchmarkError = Parse(tc.text)
			}
		})
		b.Run(tc.name+"/bytes", func(b *testing.B) {
			data := []byte(tc.text)
			b.ReportAllocs()
			for b.Loop() {
				benchmarkID, benchmarkError = ParseBytes(data)
			}
		})
	}
}

func BenchmarkFormat(b *testing.B) {
	for _, id := range []ID{0, 175928847299117063, 1412345678901234567, ^ID(0)} {
		b.Run(id.String()+"/String", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkString = id.String()
			}
		})
		b.Run(id.String()+"/AppendText", func(b *testing.B) {
			buf := make([]byte, 0, 32)
			b.ReportAllocs()
			for b.Loop() {
				benchmarkBytes, benchmarkError = id.AppendText(buf[:0])
			}
		})
		b.Run(id.String()+"/MarshalText", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkBytes, benchmarkError = id.MarshalText()
			}
		})
		b.Run(id.String()+"/MarshalJSON", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkBytes, benchmarkError = id.MarshalJSON()
			}
		})
	}
}

func BenchmarkDecodeJSON(b *testing.B) {
	for _, tc := range []struct {
		name string
		text string
	}{
		{
			name: "quoted",
			text: `"175928847299117063"`,
		},
		{
			name: "number",
			text: `175928847299117063`,
		},
		{
			name: "max",
			text: `"18446744073709551615"`,
		},
		{
			name: "escaped",
			text: `"\u003175928847299117063"`,
		},
		{
			name: "null",
			text: `null`,
		},
		{
			name: "invalid",
			text: `"17592884729911706x"`,
		},
		{
			name: "overflow",
			text: `18446744073709551616`,
		},
	} {
		data := []byte(tc.text)
		b.Run(tc.name+"/direct", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkError = benchmarkID.UnmarshalJSON(data)
			}
		})
		b.Run(tc.name+"/standard", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkError = json.Unmarshal(data, &benchmarkID)
			}
		})
	}
}

func BenchmarkJSON(b *testing.B) {
	object := benchmarkObject{
		ID:        175928847299117063,
		ChannelID: 1412345678901234567,
		IDs:       []ID{0, 175928847299117063, 1412345678901234567, ^ID(0)},
	}
	objectData, _ := json.Marshal(object)
	arrayData, _ := json.Marshal(object.IDs)
	b.Run("MarshalObject", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkBytes, benchmarkError = json.Marshal(object)
		}
	})
	b.Run("UnmarshalObject", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var dst benchmarkObject
			benchmarkError = json.Unmarshal(objectData, &dst)
		}
	})
	b.Run("MarshalArray", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkBytes, benchmarkError = json.Marshal(object.IDs)
		}
	})
	b.Run("UnmarshalArray", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var dst []ID
			benchmarkError = json.Unmarshal(arrayData, &dst)
		}
	})
}

func BenchmarkSQL(b *testing.B) {
	for _, tc := range []struct {
		name string
		src  any
	}{
		{
			name: "string",
			src:  "175928847299117063",
		},
		{
			name: "bytes",
			src:  []byte("175928847299117063"),
		},
		{
			name: "int64",
			src:  int64(175928847299117063),
		},
		{
			name: "nil",
			src:  nil,
		},
		{
			name: "negative",
			src:  int64(-1),
		},
		{
			name: "unsupported",
			src:  float64(1),
		},
	} {
		b.Run("Scan/"+tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkError = benchmarkID.Scan(tc.src)
			}
		})
	}
	b.Run("Value", func(b *testing.B) {
		id := ID(175928847299117063)
		b.ReportAllocs()
		for b.Loop() {
			benchmarkValue, benchmarkError = id.Value()
		}
	})
}

func BenchmarkExtract(b *testing.B) {
	id := ID(175928847299117063)
	b.Run("Time", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkTime = id.Time()
		}
	})
	b.Run("Parts", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkParts = uint64(id.WorkerID()) + uint64(id.ProcessID()) + uint64(id.Increment())
		}
	})
}
