package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	rqlobj "github.com/paulstuart/rqlobj"
)

var (
	logger io.Writer
	url    string
	port   = 4001
	host   = "localhost"
	debug  = false
)

func init() {
	flag.StringVar(&host, "host", host, "host to connect to")
	flag.IntVar(&port, "port", port, "port to connect to")
	flag.BoolVar(&debug, "debug", false, "enable debug tracing")
	url = fmt.Sprintf("http://%s:%d", host, port)
}

func main() {
	flag.Parse()
	url = fmt.Sprintf("http://%s:%d", host, port)
	fmt.Println("URL IS:", url)
	myList(url)
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

const testTable = `rdbms_structs`

const createdb = `
drop table if exists ` + testTable + `;

create table if not exists ` + testTable + ` (
id integer primary key,
name text,
kind integer,
data text,
ts DATETIME DEFAULT (datetime('now','utc')),
ts2 integer DEFAULT (datetime('now','utc'))
);

create table if not exists test01 (
id integer primary key,
testname text,
testkind integer,
testdata text
);

create table if not exists test02 (
id integer primary key,
testname text,
testkind integer,
testdata text,
ts DATETIME DEFAULT (datetime('now','utc')),
ts2 integer DEFAULT (datetime('now','utc')),
ts3 DATE    DEFAULT (datetime('now','utc')),
ts4 TEXT    DEFAULT (datetime('now','utc')),
ts5 NUMERIC DEFAULT (datetime('now','utc'))
);

insert into test01 (testname, testkind, testdata)
       values ('booboo', 23, "easter egg");
insert into test02 (testname, testkind, testdata)
       values ('booboo', 21, "yeastnog");
insert into test02 (testname, testkind, testdata)
       values ('booboo', 20, "yeastnog");
insert into test02 (testname, testkind, testdata)
       values ('booboo', 24, "yeastnog");
`

func myList(url string) {
	var trace io.Writer
	if debug {
		trace = os.Stdout
	}
	dbu, err := rqlobj.NewRqlite(url, logger, trace)
	if err != nil {
		log.Fatalf("NEW URL:%s err:%v", url, err)
	}
	//if results, err := dbu.Write(strings.Split(createdb, ";")...); err != nil {
	queries := strings.Split(createdb, ";")
	/*
		for i, query := range queries {
			queries[i] = strings.TrimSpace(query) + ";"
		}
	*/
	if results, err := dbu.Write(queries...); err != nil {
		for i, result := range results {
			if result.Err != nil {
				log.Printf("WRITE QUERY: %s\nERROR: %+v\n", queries[i], result.Err)
			}
		}
		log.Fatalf("WRITE ERR: %+v\n", err)
	}
	self := &testStruct{
		Name:      "Bobby",
		Kind:      123,
		Data:      "hey now",
		Timestamp: time.Now(),
		When:      time.Now().Add(time.Hour * -24),
	}
	if err := dbu.Add(self); err != nil {
		log.Fatal(err)
	}
	soft := &testStruct{
		Name:      "Betty",
		Kind:      42,
		Data:      "hello world",
		Timestamp: time.Now(),
	}
	if err := dbu.Add(soft); err != nil {
		log.Fatal(err)
	}
	var list _testStruct
	if err := dbu.List(&list); err != nil {
		log.Fatalf("LIST ERR: %+v\n", err)
	}
	for i, v := range list {
		fmt.Printf("%d: %+v\n", i, v)
	}
}
