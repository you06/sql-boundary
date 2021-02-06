package main

import (
	"database/sql"
	"flag"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var (
	dsn string
)

func init() {
	flag.StringVar(&dsn, "dsn", "root:@tcp(127.0.0.1:4000)/test", "dsn")
	flag.Parse()
}

func main() {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	MustExec(db, "SET SESSION time_zone = 'UTC'")
	caseCount := 0
	if err := IterateCases(func(fName string, one *Case) error {
		MustExec(db, "DROP TABLE IF EXISTS t")
		err := one.Execute(db, "t")
		caseCount++
		return err
	}, true); err != nil {
		fmt.Println("test failed", err)
	} else {
		fmt.Println(caseCount, "cases passed")
	}
}

func MustExec(db *sql.DB, sqlStmt string) {
	_, err := db.Exec(sqlStmt)
	if err != nil {
		panic(err)
	}
}
