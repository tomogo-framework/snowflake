package snowflake

import "time"

// MinID returns the inclusive minimum numeric ID in the millisecond containing
// t, with all 22 lower bits zero. Submillisecond precision is rounded down.
// The accepted interval is [Epoch, Epoch + 2^42 milliseconds); times outside it
// return zero and ErrTimeRange. Location and monotonic clock readings do not
// affect the result. The result is a boundary, not a newly issued Discord ID.
//
// Discord's exclusive before=MinID(t) cursor excludes the entire containing
// millisecond. For after pagination including that millisecond, use MinID(t)-1
// only if the result is nonzero; zero has no preceding representable ID. Follow
// each endpoint's cursor rules, including treatment of zero or absent cursors.
func MinID(t time.Time) (ID, error) {
	// Compare instants before calling UnixMilli, which can overflow for extreme
	// time.Time values outside Discord's much smaller timestamp range.
	if t.Before(time.UnixMilli(Epoch)) || !t.Before(time.UnixMilli(Epoch+(1<<42))) {
		return 0, ErrTimeRange
	}

	return ID(t.UnixMilli()-Epoch) << 22, nil
}

// MaxID returns the inclusive maximum numeric ID in the millisecond containing
// t, with all 22 lower bits set. It uses MinID's rounding and range rules.
// The result is a boundary, not a newly issued Discord ID.
//
// Discord's exclusive after=MaxID(t) cursor excludes the entire containing
// millisecond. For before pagination including that millisecond, use MaxID(t)+1
// only if the result is below the maximum uint64; that maximum has no following
// representable ID. Follow each endpoint's cursor rules.
func MaxID(t time.Time) (ID, error) {
	id, err := MinID(t)
	if err != nil {
		return 0, err
	}

	return id | ((1 << 22) - 1), nil
}
