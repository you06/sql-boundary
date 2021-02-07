package main

import (
	"fmt"
	"time"
)

var (
	// TIME_MIN = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	TIME_MIN  = time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)
	TIME_MAX  = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	TS_MIN    = time.Date(1970, 1, 1, 0, 0, 1, 0, time.UTC)
	TS_MAX    = time.Date(2038, 1, 19, 3, 14, 7, 0, time.UTC)
	timeUnits = []string{"days", "hours", "minutes", "seconds"}
)

type MySQLDuration time.Duration

func (d MySQLDuration) String() string {
	var (
		s        string
		minus    = false
		days     time.Duration
		hours    time.Duration
		minutes  time.Duration
		seconds  time.Duration
		duration = time.Duration(d)
	)
	if duration < 0 {
		duration = -duration
		minus = true
	}
	days = duration / (24 * time.Hour)
	duration = duration - days*24*time.Hour
	hours = duration / time.Hour
	duration = duration - hours*time.Hour
	minutes = duration / time.Minute
	duration = duration - minutes*time.Minute
	seconds = duration / time.Second
	if days != 0 {
		s = fmt.Sprintf("%d ", days)
	}
	s = fmt.Sprintf("%s%d:%d:%d", s, hours, minutes, seconds)
	if minus {
		return "'-" + s + "'"
	}
	return "'" + s + "'"
}

type Date time.Time

func (d Date) String() string {
	return "'" + time.Time(d).Format("2006-01-02") + "'"
}

type Datetime time.Time

func (d Datetime) String() string {
	return "'" + time.Time(d).Format("2006-01-02 15:04:05") + "'"
}

type Timestamp time.Time

func (t Timestamp) String() string {
	return "'" + time.Time(t).Format("2006-01-02 15:04:05") + "'"
}

type Days int

func (d Days) String() string {
	return fmt.Sprintf("INTERVAL %d DAY", d)
}

type Hours int

func (h Hours) String() string {
	return fmt.Sprintf("INTERVAL %d HOUR", h)
}

type Minutes int

func (m Minutes) String() string {
	return fmt.Sprintf("INTERVAL %d MINUTE", m)
}

type Seconds int

func (s Seconds) String() string {
	return fmt.Sprintf("INTERVAL %d SECOND", s)
}

type dateCase struct {
	valid bool
	t     time.Time
	d     time.Duration
}

var (
	dateTypes = []struct {
		min time.Time
		max time.Time
		tp  string
	}{
		{min: TS_MIN, max: TS_MAX, tp: "timestamp"},
		{min: TIME_MIN, max: TIME_MAX, tp: "date"},
		{min: TIME_MIN, max: TIME_MAX, tp: "datetime"},
	}
)

// DateFunction is used for generate date related function bound conditions
// TL;DR. Use one.t + ond.d and one.t - (-one.d). eg. adddate(ont.t, one.d), subdate(ont.t, -one,d)
// it combinates cases follow this rule,
// for each result, add result.d and sub -result.d should get same effect
// if valid is true, it means this combinator is definitly valid
// may invalid cases   valid cases
// max + (+ ...)       max + (- ...)
// min + (- ...)       min + (+ ...)
// max - (- ...)       max - (+ ...)
// min - (+ ...)       min - (- ...)
func DateFunction(fn func(one dateCase, dataDef string)) {
	for _, one := range dateTypes {
		DateFunctionOne(one.min, one.max, one.tp, fn)
	}
}

func DateFunctionOne(min, max time.Time, dataDef string, fn func(one dateCase, dataDef string)) {
	combinate := func(min, max time.Time, d time.Duration) []struct {
		valid bool
		t     time.Time
		d     time.Duration
	} {
		r := make([]struct {
			valid bool
			t     time.Time
			d     time.Duration
		}, 4)
		r[0].valid = false
		r[0].t = max
		r[0].d = d
		r[1].valid = false
		r[1].t = min
		r[1].d = -d
		r[2].valid = true
		r[2].t = max
		r[2].d = -d
		r[3].valid = true
		r[3].t = min
		r[3].d = d
		// Sometimes date and datetime does not report error when the lower bound is exceeded
		if dataDef != "timestamp" {
			return append(r[:1], r[2:]...)
		}
		return r
	}

	safeDuration := 24 * time.Hour
	tsMinValid, tsMaxValid := min.Add(safeDuration), max.Add(-safeDuration)

	// addDateFunc, subDateFunc := GetFunc(ast.AddDate), GetFunc(ast.SubDate)
	// addTimeFunc, subTimeFunc := GetFunc(ast.AddTime), GetFunc(ast.SubTime)

	for i := 0; i <= 3; i++ {
		d := time.Duration(i) * safeDuration
		valid := d <= safeDuration
		for _, one := range combinate(tsMinValid, tsMaxValid, d) {
			one.valid = valid || one.valid
			fn(one, dataDef)
		}
	}
}

func parseTime(t time.Time, dataDef string) fmt.Stringer {
	switch dataDef {
	case "timestamp":
		return Timestamp(t)
	case "date":
		return Date(t)
	case "datetime":
		return Datetime(t)
	default:
		panic("unknown datetype " + dataDef)
	}
}

// parseDurations translate duration to specific format
func parseDuration(d time.Duration, format string) fmt.Stringer {
	switch format {
	case "days":
		return Days(d / (24 * time.Hour))
	case "hours":
		return Hours(d / time.Hour)
	case "minutes":
		return Minutes(d / time.Minute)
	case "seconds":
		return Seconds(d / time.Second)
	default:
		panic("unknown datetype " + format)
	}
}
