# snowflake

A small Go library for Discord snowflakes. Standard library only, with no ID
generation or Discord framework dependency. Requires Go 1.27.0 or newer.

```sh
go get github.com/tomogo-framework/snowflake
```

```go
import "github.com/tomogo-framework/snowflake"

id, err := snowflake.Parse("175928847299117063")
if err != nil {
    return err
}
fmt.Println(id.String())                        // 175928847299117063
fmt.Println(id.Time().Format(time.RFC3339Nano)) // 2016-04-30T11:18:25.796Z
fmt.Println(id.WorkerID(), id.ProcessID(), id.Increment()) // 1 0 7
```

`ID` is a `uint64`, so it is comparable, sortable with ordinary comparisons, and
usable as a map key. Zero means absent by convention; it formats as `0` and its
timestamp is Discord's epoch, not Go's zero time. Parsing accepts every `uint64`
value. It does not prove that Discord issued an ID or that the ID exists.

## Parsing and errors

`Parse(string)` and `ParseBytes([]byte)` accept one or more ASCII decimal digits.
Zero and leading zeros are valid. Empty input, signs, surrounding whitespace,
non-ASCII digits, malformed UTF-8, fractions, exponents, and overflow are rejected.
`MustParse` panics on invalid input and is useful for known constants in fixtures.

Inspect errors with `errors.Is`; error messages never contain the input:

| Error            | Meaning                                                  |
|------------------|----------------------------------------------------------|
| `ErrSyntax`      | Invalid decimal input or unsupported JSON representation |
| `ErrRange`       | Decimal overflow or a negative SQL integer               |
| `ErrNilReceiver` | A decoder or scanner was called on a nil `*ID`           |
| `ErrSource`      | Unsupported SQL source type                              |
| `ErrTimeRange`   | Time outside the timestamp range                         |

Failed parsing returns zero and an error. Failed `UnmarshalText`, `UnmarshalJSON`,
or `Scan` leaves the destination unchanged. Those methods return `ErrNilReceiver`
for a nil receiver regardless of input. Value-receiver methods require a value;
calling them through a nil pointer panics under Go's normal method rules.

## JSON and text

```go
type Message struct {
    ID        snowflake.ID `json:"id"`
    ChannelID snowflake.ID `json:"channel_id"`
    ParentID  snowflake.ID `json:"parent_id,omitempty"`
}
```

`json.Marshal` emits quoted decimal strings, preserving exact precision through
`18446744073709551615`. Use ordinary tags, without `,string`. Zero fields may be
omitted with `omitempty`. Map keys also work with `encoding/json`.

`json.Unmarshal` and direct `UnmarshalJSON` calls accept quoted decimal strings,
nonnegative integer numbers, and `null`, which resets an existing ID to zero.
Standard JSON whitespace around the value is allowed. Valid string escapes are
decoded, so `"\u0034\u0032"` becomes 42. Decoded strings follow the plain decimal
policy: `"00042"` is valid, but the bare JSON number `00042` is invalid. Numeric
`-0`, fractions, and exponents are rejected even when mathematically integral.

`encoding/json` can return its own errors before invoking the library. Preservation
applies to each ID receiver, not an entire containing struct or map: ordinary Go
JSON decoding can update other fields before finding an error. A nil `*ID` marshals
as `null` through `encoding/json`; JSON null into an `**ID` follows Go's pointer
rules and sets the pointer to nil.

`MarshalText` and `UnmarshalText` use unquoted decimal text. `AppendText` implements
`encoding.TextAppender` and supports caller-owned buffers:

```go
buf := make([]byte, 0, 32)
buf = append(buf, "id="...)
buf, _ = id.AppendText(buf) // The error is always nil.
// Reuse buf[:0] for the next operation.
```

Appending preserves the prefix and reuses sufficient capacity. Parsing and
decoding do not retain or modify input bytes. Marshal methods return owned buffers.
Independent values can be used concurrently; synchronize writes to a shared ID
or buffer. The package has no shared mutable state.

## Databases

`ID` implements `driver.Valuer` and `*ID` implements `sql.Scanner`:

```go
var id snowflake.ID
err := db.QueryRowContext(ctx, "SELECT discord_id FROM users WHERE name = ?", name).Scan(&id)
// Use the placeholder syntax required by your database driver.
```

Pass an ID directly as a query argument. `Value` returns a decimal string, including
`"0"` for zero. Choose text or a sufficiently wide exact numeric column such as
`DECIMAL(20,0)`. A signed `BIGINT` cannot hold the complete range; text ordering is
lexicographic rather than numeric. Driver/database coercion still depends on the
schema.

`Scan` accepts `string`, `[]byte`, nonnegative `int64`, and `nil`. SQL NULL resets
the value to zero, so use a separate nullable representation when NULL and zero
must remain distinct. A typed nil byte slice is empty text and is rejected.
Other source types, including `uint64`, `float64`, and `json.Number`, are rejected.

## Pagination boundaries

`MinID(t)` and `MaxID(t)` return the inclusive minimum and maximum numeric ID in
the millisecond containing `t`. They round down, discarding submillisecond precision.
Locations and monotonic clock readings do not affect the result. The accepted
interval is `[2015-01-01T00:00:00Z, time.UnixMilli(5818116911104))`.
Times before the epoch or at/after the exclusive upper endpoint return `ErrTimeRange`.

```go
start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
before, err := snowflake.MinID(start)
if err != nil {
    return err
}
query.Set("before", before.String()) // IDs strictly earlier than start's millisecond.
```

For Discord endpoints with exclusive `before`/`after` cursors:

| Cursor              | Numeric result set                                               |
|---------------------|------------------------------------------------------------------|
| `before=MinID(t)`   | Milliseconds earlier than t's containing millisecond             |
| `after=MaxID(t)`    | Milliseconds later than t's containing millisecond               |
| `after=MinID(t)-1`  | Includes t's entire millisecond, only when MinID(t) > 0          |
| `before=MaxID(t)+1` | Includes t's entire millisecond, only when MaxID(t) < max uint64 |

Never subtract from zero or add to max uint64. Endpoint restrictions and treatment
of zero/absent cursors still apply. For example, Get Channel Messages permits only
one of `before`, `after`, or `around` per request. These values are pagination
boundaries, not newly issued or globally unique Discord IDs.

Format and serialization verified on **2026-09-07 (America/Chicago)** against
[Discord's snowflake reference](https://docs.discord.com/developers/reference#snowflakes),
[serialization rules](https://docs.discord.com/developers/reference#id-serialization),
and [message pagination](https://docs.discord.com/developers/resources/message#get-channel-messages).

## Performance and validation

Benchmarks live in `benchmark_test.go`. They cover 18/19/20-digit IDs, zero,
malformed input, overflow, parsing from strings/bytes, formatting and buffer reuse,
direct and standard JSON decoding, multi-ID JSON objects/arrays, SQL, and extraction.

```sh
go test -run '^$' -bench . -benchmem -benchtime=100ms -count=10 -cpu=1
```

Each workload runs ten times with allocation reporting and one benchmark CPU.
Use `benchstat` to summarize the samples. Measured medians on Go 1.27.1,
Windows/amd64, Intel i7-9700K (2026-09-07):

| Workload                             | ns/op | B/op | allocs/op |
|--------------------------------------|------:|-----:|----------:|
| Parse 18-digit bytes                 | 45.16 |    0 |         0 |
| Append 18-digit text                 | 20.75 |    0 |         0 |
| Direct JSON marshal, 18 digits       | 49.53 |   24 |         1 |
| Direct quoted JSON decode            | 52.13 |    0 |         0 |
| SQL scan int64                       |  3.61 |    0 |         0 |
| Standard JSON marshal, six-ID object |  1257 |  384 |         9 |
| Standard JSON decode, six-ID object  |  1553 |  104 |         4 |

The append benchmark reuses a buffer with sufficient capacity. Escaped JSON
strings use the standard JSON decoder. Timings vary with hardware, toolchain,
and machine activity; these measurements describe the listed library workloads
only, not end-to-end bot performance. Tests provide 100% statement coverage.

Run `go test ./...`, `go test -race ./...`, and `go vet ./...` locally. CI covers
Linux, Windows, and macOS on Go 1.27.0 and stable, plus staticcheck, govulncheck,
32-bit tests, and bounded fuzzing. Runnable examples and four fuzz targets are
included. Use `go test -run '^$' -fuzz '^FuzzUnmarshalJSON$' -fuzztime=15s` to fuzz
one target at a time.
