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
	caseFailed := 0
	IterateCases(func(fName string, one *Case) {
		MustExec(db, "DROP TABLE IF EXISTS t")
		err := one.Execute(db, "t")
		if err != nil {
			fmt.Println(err)
			fmt.Println()
			caseFailed++
		}
		caseCount++
	})
	fmt.Printf("%d cases passed, %d failed\n", caseCount, caseFailed)
}

func MustExec(db *sql.DB, sqlStmt string) {
	_, err := db.Exec(sqlStmt)
	if err != nil {
		panic(err)
	}
}
