package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	rqlobj "github.com/paulstuart/rqlobj"
)

//const URL = "http://localhost:4001"

var (
	logger io.Writer
	URL    string
	port   int = 4001
	host       = "localhost"
	debug      = false
)

func init() {
	flag.StringVar(&host, "host", host, "host to connect to")
	flag.IntVar(&port, "port", port, "port to connect to")
	flag.BoolVar(&debug, "debug", false, "enable debug tracing")
	URL = fmt.Sprintf("http://%s:%d", host, port)
}

func main() {
	flag.Parse()
	URL = fmt.Sprintf("http://%s:%d", host, port)
	//URL = fmt.Sprintf("http://%s:%d", *hostFlag, *portFlag)
	URL = fmt.Sprintf("http://%s:%d", host, port)
	fmt.Println("URL IS:", URL)
	myList(URL)
	return

	/*
		conn, err := gorqlite.Open("http://localhost:4001")
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
	*/
}

func myList(url string) {
	var trace io.Writer
	if debug {
		trace = os.Stdout
	}
	dbu, err := rqlobj.NewRqlite(URL, logger, trace)
	if err != nil {
		log.Fatalf("URL:%s err:%v", URL, err)
	}
	var list _testStruct
	if err := dbu.List(&list); err != nil {
		log.Fatal(err)
	}
	for i, v := range list {
		fmt.Printf("%d: %+v\n", i, v)
	}
}
