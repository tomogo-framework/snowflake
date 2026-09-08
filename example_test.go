package snowflake_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tomogo-framework/snowflake"
)

func ExampleParse() {
	id, err := snowflake.Parse("175928847299117063")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(id)
	fmt.Println(id.Time().Format(time.RFC3339Nano))
	fmt.Println(id.WorkerID(), id.ProcessID(), id.Increment())
	// Output:
	// 175928847299117063
	// 2016-04-30T11:18:25.796Z
	// 1 0 7
}

func ExampleParseBytes() {
	id, err := snowflake.ParseBytes([]byte("00042"))
	fmt.Println(id, err)
	// Output: 42 <nil>
}

func ExampleID_AppendText() {
	id := snowflake.ID(175928847299117063)
	buf := make([]byte, 0, 32)
	buf = append(buf, "id="...)
	buf, _ = id.AppendText(buf)
	fmt.Println(string(buf))
	// Output: id=175928847299117063
}

func ExampleID_MarshalJSON() {
	type message struct {
		ID        snowflake.ID `json:"id"`
		ChannelID snowflake.ID `json:"channel_id"`
	}
	var m message
	err := json.Unmarshal([]byte(`{"id":"175928847299117063","channel_id":42}`), &m)
	if err != nil {
		fmt.Println(err)
		return
	}

	data, _ := json.Marshal(m)
	fmt.Println(string(data))
	// Output: {"id":"175928847299117063","channel_id":"42"}
}

func ExampleID_Scan() {
	var id snowflake.ID
	if err := id.Scan("18446744073709551615"); err != nil {
		fmt.Println(err)
		return
	}

	value, _ := id.Value()
	fmt.Printf("%T: %v\n", value, value)
	// Output: string: 18446744073709551615
}

func ExampleMinID() {
	t := time.Date(2015, 1, 1, 0, 0, 0, 1500000, time.UTC)
	min, _ := snowflake.MinID(t)
	max, _ := snowflake.MaxID(t)
	fmt.Println(min, max)
	fmt.Println(min.Time().Format(time.RFC3339Nano))
	// before=min excludes this entire millisecond and all later IDs.
	// after=max excludes this entire millisecond and all earlier IDs.
	// Output:
	// 4194304 8388607
	// 2015-01-01T00:00:00.001Z
}
