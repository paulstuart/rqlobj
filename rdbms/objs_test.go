package main

import (
	"fmt"
	//	"reflect"
	"io"
	"os"
	"testing"
	"time"

	"github.com/paulstuart/rqlobj"
)

const createdb = `
drop table if exists structs;

create table if not exists structs (
id integer primary key,
name text,
kind integer,
ts2 integer DEFAULT (datetime('now','localtime')),
-- integer DEFAULT 23,
data text,
astring string,
ts DATE DEFAULT (datetime('now','localtime')),
--ts timestamp default (strftime('%s', 'now')
anint integer
);
`

const timing = `
create table if not exists timing (
id integer primary key,
name text,
kind integer,
--when DATE DEFAULT (datetime('now','localtime'))
);

`

const inserted = `
insert into paultest (name) values('Ringo');
`

func TestCreate(t *testing.T) {
	var trace io.Writer
	if debug {
		trace = os.Stdout
	}
	dbu, err := rqlobj.NewRqlite(URL, trace)
	if err != nil {
		t.Fatalf("URL:%s err:%v", URL, err)
	}
	query := createdb
	_, err = dbu.Write(query)
	if err != nil {
		fmt.Printf("OOPSIE:%s\n\n", query)
		t.Fatal(err)
	}
	self := &testStruct{
		Name: "Bobby",
		Kind: 123,
		//Data: []byte("hey now"),
		Data:      "hey now",
		Timestamp: time.Now(),
		//Timestamp: time.Now().Unix(),
	}
	if err := dbu.Add(self); err != nil {
		t.Fatal(err)
	}
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
	fmt.Println("SAVED!")
}

func TestList(t *testing.T) {
	var trace io.Writer
	if debug {
		trace = os.Stdout
	}
	dbu, err := rqlobj.NewRqlite(URL, trace)
	if err != nil {
		t.Fatalf("URL:%s err:%v", URL, err)
	}
	var list _testStruct
	if err := dbu.List(&list); err != nil {
		t.Fatal(err)
	}
	//fmt.Println("LIST:", list)
	for i, v := range list {
		fmt.Printf("%d: %+v\n", i, v)
	}
}
