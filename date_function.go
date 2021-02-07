package main

import "github.com/pingcap/parser/ast"

func (c Cases) TestAddDateFunc() {
	fn := GetFunc(ast.AddDate)
	DateFunction(func(one dateCase, dataDef string) {
		for _, unit := range timeUnits {
			c := NewCase(one.valid, dataDef, parseTime(one.t, dataDef), Column{}, parseDuration(one.d, unit))
			fn.Cases = append(fn.Cases, c)
		}
	})
}

func (c Cases) TestSubDateFunc() {
	fn := GetFunc(ast.SubDate)
	DateFunction(func(one dateCase, dataDef string) {
		for _, unit := range timeUnits {
			c := NewCase(one.valid, dataDef, parseTime(one.t, dataDef), Column{}, parseDuration(-one.d, unit))
			fn.Cases = append(fn.Cases, c)
		}
	})
}

func (c Cases) TestAddTimeFunc() {
	fn := GetFunc(ast.AddTime)
	DateFunction(func(one dateCase, dataDef string) {
		c := NewCase(one.valid, dataDef, parseTime(one.t, dataDef), Column{}, MySQLDuration(one.d))
		fn.Cases = append(fn.Cases, c)
	})
}

func (c Cases) TestSubTimeFunc() {
	fn := GetFunc(ast.SubTime)
	DateFunction(func(one dateCase, dataDef string) {
		c := NewCase(one.valid, dataDef, parseTime(one.t, dataDef), Column{}, MySQLDuration(-one.d))
		fn.Cases = append(fn.Cases, c)
	})
}
