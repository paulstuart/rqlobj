package main

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/paulstuart/rqlobj"
)

const test_table = `rdbms_structs`

const createdb = `
drop table if exists ` + test_table + `;

create table if not exists ` + test_table + ` (
id integer primary key,
name text,
kind integer,
data text,
ts DATETIME DEFAULT (datetime('now','utc')),
ts2 integer DEFAULT (datetime('now','utc'))
);
`

func TestCreate(t *testing.T) {
	var trace io.Writer
	if debug {
		trace = os.Stdout
	}
	dbu, err := rqlobj.NewRqlite(URL, logger, trace)
	if err != nil {
		t.Fatalf("URL:%s err:%v", URL, err)
	}
	query := createdb
	_, err = dbu.Write(query)
	if err != nil {
		t.Fatalf("query:%q error:%v", query, err)
	}
	self := &testStruct{
		Name:      "Bobby",
		Kind:      123,
		Data:      "hey now",
		Timestamp: time.Now(),
		When:      time.Now().Add(time.Hour * -24),
	}
	if err := dbu.Add(self); err != nil {
		t.Fatal(err)
	}
	return
	const layout = "2006-01-02 15:04:05"
	const before = "2001-09-10 11:11:11"
	when, _ := time.Parse(layout, before)
	other := &testStruct{
		Name: "Betty",
		Kind: 99,
		Data: "woo hoo",
		When: when,
	}
	if err := dbu.Add(other); err != nil {
		t.Fatal(err)
	}
}

func TestList(t *testing.T) {
	var trace io.Writer
	if debug {
		trace = os.Stdout
	}
	dbu, err := rqlobj.NewRqlite(URL, logger, trace)
	if err != nil {
		t.Fatalf("URL:%s err:%v", URL, err)
	}
	var list _testStruct
	if err := dbu.ListQuery(&list, "limit 5"); err != nil {
		t.Fatal(err)
	}
	for i, v := range list {
		t.Logf("%d: %+v\n", i, v)
	}
}
