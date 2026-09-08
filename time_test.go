package snowflake_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/tomogo-framework/snowflake"
)

func TestTimeBounds(t *testing.T) {
	for _, tc := range []struct {
		name string
		time time.Time
		min  snowflake.ID
		max  snowflake.ID
	}{
		{
			name: "epoch",
			time: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
			min:  0,
			max:  4194303,
		},
		{
			name: "floor",
			time: time.Date(2015, 1, 1, 0, 0, 0, 999999, time.UTC),
			min:  0,
			max:  4194303,
		},
		{
			name: "next",
			time: time.Date(2015, 1, 1, 0, 0, 0, 1000000, time.UTC),
			min:  4194304,
			max:  8388607,
		},
		{
			name: "example",
			time: time.Date(2016, 4, 30, 11, 18, 25, 796999999, time.UTC),
			min:  175928847298985984,
			max:  175928847303180287,
		},
		{
			name: "zone",
			time: time.Date(2014, 12, 31, 18, 0, 0, 0, time.FixedZone("CST", -6*3600)),
			min:  0,
			max:  4194303,
		},
		{
			name: "last",
			time: time.Unix(5818116911, 103999999),
			min:  18446744073705357312,
			max:  math.MaxUint64,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			min, minErr := snowflake.MinID(tc.time)
			max, maxErr := snowflake.MaxID(tc.time)
			if minErr != nil || maxErr != nil || min != tc.min || max != tc.max {
				t.Fatalf("bounds = %v, %v (%v, %v); want %v, %v", min, max, minErr, maxErr, tc.min, tc.max)
			}

			if min.Time().UnixMilli() != tc.time.UnixMilli() || max.Time().UnixMilli() != tc.time.UnixMilli() {
				t.Fatal("inclusive bounds have wrong timestamp")
			}

			if min > 0 && (min-1).Time().UnixMilli() != tc.time.UnixMilli()-1 {
				t.Fatal("before boundary has wrong timestamp")
			}

			if max < math.MaxUint64 && (max+1).Time().UnixMilli() != tc.time.UnixMilli()+1 {
				t.Fatal("after boundary has wrong timestamp")
			}
		})
	}

	for _, instant := range []time.Time{
		{},
		time.Unix(1420070399, 999999999),
		time.Unix(5818116911, 104000000),
		time.UnixMilli(math.MinInt64),
		time.UnixMilli(math.MaxInt64),
		time.Date(-1000000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(1000000, 1, 1, 0, 0, 0, 0, time.UTC),
	} {
		for _, bound := range []func(time.Time) (snowflake.ID, error){snowflake.MinID, snowflake.MaxID} {
			id, err := bound(instant)
			if id != 0 || !errors.Is(err, snowflake.ErrTimeRange) {
				t.Errorf("bound(%v) = %v, %v", instant, id, err)
			}
		}
	}

	now := time.Now()
	withMonotonic, _ := snowflake.MinID(now)
	withoutMonotonic, _ := snowflake.MinID(now.Round(0))
	if withMonotonic != withoutMonotonic {
		t.Fatal("monotonic clock affects boundary")
	}
}
