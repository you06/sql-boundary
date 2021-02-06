package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pingcap/parser/ast"
)

// Func ...
type Func struct {
	Name  string
	Cases []*Case
}

// Column is a special arg
type Column struct{}

func (c Column) String() string {
	return "c"
}

// Case ...
type Case struct {
	FName   string
	Valid   bool
	DataDef string
	InitVal fmt.Stringer
	Args    []fmt.Stringer
}

func NewCase(valid bool, dataDef string, initVal fmt.Stringer, args ...fmt.Stringer) *Case {
	return &Case{
		Valid:   valid,
		DataDef: dataDef,
		InitVal: initVal,
		Args:    args,
	}
}

func (c Case) Execute(db *sql.DB, table string) error {
	argStrings := make([]string, len(c.Args))
	for i, arg := range c.Args {
		argStrings[i] = arg.String()
	}

	var (
		err       error
		createSQL = "CREATE TABLE t(c " + c.DataDef + ");"
		insertSQL = "INSERT INTO t VALUES(" + c.InitVal.String() + ");"
		updateSQL = fmt.Sprintf("UPDATE t SET c = %s(%s);", c.FName, strings.Join(argStrings, ","))
	)
	// insert and update
	// MustExec(db, "TRUNCATE TABLE t")
	_, err = db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("exec SQL err: %s, err: %v", createSQL, err)
	}
	_, err = db.Exec(insertSQL)
	if err != nil {
		return fmt.Errorf("exec SQL err: %s, err: %v", insertSQL, err)
	}
	_, err = db.Exec(updateSQL)
	if (err == nil) != c.Valid {
		return fmt.Errorf("%v failed, err: %v\nReproduce:\n%v", c, err, strings.Join([]string{
			createSQL,
			insertSQL,
			updateSQL,
		}, "\n"))
	}
	return nil
}

var funcs = make(map[string]*Func)

func GetFunc(name string) *Func {
	f, ok := funcs[name]
	if ok {
		return f
	}
	funcs[name] = &Func{Name: name}
	return funcs[name]
}

func IterateCases(call func(fName string, one *Case)) {
	for _, f := range funcs {
		for _, one := range f.Cases {
			call(f.Name, one)
		}
	}
}

func init() {
	InitDateFunctions()
	// assign func names
	IterateCases(func(fName string, one *Case) {
		one.FName = fName
	})
}

var (
	// TIME_MIN = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	TIME_MIN = time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)
	TIME_MAX = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	TS_MIN   = time.Date(1970, 1, 1, 0, 0, 1, 0, time.UTC)
	TS_MAX   = time.Date(2038, 1, 19, 3, 14, 7, 0, time.UTC)
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

func InitDateFunctions() {
	DateFunctions(TS_MIN, TS_MAX, "timestamp")
	DateFunctions(TIME_MIN, TIME_MAX, "date")
	DateFunctions(TIME_MIN, TIME_MAX, "datetime")
}

func DateFunctions(min, max time.Time, dataDef string) {
	parseTime := func(t time.Time) fmt.Stringer {
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
	parseDuration := func(d time.Duration, format string) fmt.Stringer {
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
	// combinate compose cases follow this rule,
	// for each result, add result.d and sub -result.d should get same effect
	// if valid is true, it means this combinator is definitly valid
	// invalid cases   valid cases
	// max + (+ ...)   max + (- ...)
	// min + (- ...)   min + (+ ...)
	// max - (- ...)   max - (+ ...)
	// min - (+ ...)   min - (- ...)
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

	addDateFunc, subDateFunc := GetFunc(ast.AddDate), GetFunc(ast.SubDate)
	addTimeFunc, subTimeFunc := GetFunc(ast.AddTime), GetFunc(ast.SubTime)

	for i := 0; i <= 3; i++ {
		d := time.Duration(i) * safeDuration
		valid := d <= safeDuration
		for _, one := range combinate(tsMinValid, tsMaxValid, d) {
			valid = valid || one.valid
			// + (+ ...)
			addTimeFunc.Cases = append(addTimeFunc.Cases, NewCase(valid, dataDef, parseTime(one.t), Column{}, MySQLDuration(one.d)))
			// - (- ...)
			subTimeFunc.Cases = append(subTimeFunc.Cases, NewCase(valid, dataDef, parseTime(one.t), Column{}, MySQLDuration(-one.d)))
			for _, f := range []string{"days", "hours", "minutes", "seconds"} {
				// + (+ ...)
				addDateFunc.Cases = append(addDateFunc.Cases, NewCase(valid, dataDef, parseTime(one.t), Column{}, parseDuration(one.d, f)))
				// - (- ...)
				subDateFunc.Cases = append(subDateFunc.Cases, NewCase(valid, dataDef, parseTime(one.t), Column{}, parseDuration(-one.d, f)))
			}
		}
	}
}

// // Func ...
// type Func struct {
// 	Name  string
// 	Cases []Case
// }

// // Case ...
// type Case struct {
// 	Positive bool
// 	InitVal  fmt.Stringer
// 	Args     []fmt.Stringer
// 	DataDef  string
// }
