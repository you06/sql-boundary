package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
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

var (
	funcs      = make(map[string]*Func)
	funcMu     sync.Mutex
	assignName int32 = 0
)

func GetFunc(name string) *Func {
	funcMu.Lock()
	defer funcMu.Unlock()
	f, ok := funcs[name]
	if ok {
		return f
	}
	funcs[name] = &Func{Name: name}
	return funcs[name]
}

func IterateCases(call func(fName string, one *Case)) {
	// assign func names when first call this function
	if atomic.CompareAndSwapInt32(&assignName, 0, 1) {
		IterateCases(func(fName string, one *Case) {
			one.FName = fName
		})
	}
	for _, f := range funcs {
		for _, one := range f.Cases {
			call(f.Name, one)
		}
	}
}

type Cases struct{}

func init() {
	var cases Cases
	casesType := reflect.ValueOf(&cases)
	for i := 0; i < casesType.NumMethod(); i++ {
		method := casesType.Method(i)
		method.Call(nil)
	}
	// assign func names
	IterateCases(func(fName string, one *Case) {
		one.FName = fName
	})
}
