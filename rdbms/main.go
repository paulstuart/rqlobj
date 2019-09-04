package main

import (
	"flag"
	"fmt"
	"log"

	rqlite "github.com/rqlite/gorqlite"
)

//const URL = "http://localhost:4001"

var (
	portFlag = flag.Int("port", 4001, "port to connect to")
	hostFlag = flag.String("host", "localhost", "host to connect to")
	URL      string
)

func main() {
	flag.Parse()
	URL = fmt.Sprintf("http://%s:%d", *hostFlag, *portFlag)
	try()
	return
	conn, err := rqlite.Open("http://localhost:4001")
	if err != nil {
		log.Fatal(err)
	}
	insert := []string{
		`insert into people values("biteme", 123)`,
		`insert into people values("hoohaw", 666)`,
	}
	inserted, err := conn.Write(insert)
	if err != nil {
		log.Fatal(err)
	}
	for _, result := range inserted {
		fmt.Printf("Written: %+v\n", result)
	}

	query := []string{"select * from clones"}
	results, err := conn.Query(query)
	if err != nil {
		panic(err)
	}
	for _, result := range results {
		for result.Next() {
			m, err := result.Map()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("\nRESULT %+v\n", m)
		}
	}
}
