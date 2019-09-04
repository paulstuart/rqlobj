package main

import (
	"fmt"
	//	"reflect"
	"time"

	"github.com/paulstuart/rqlobj"
)

type testStruct struct {
	ID       int64     `sql:"id" key:"true" table:"structs"`
	Name     string    `sql:"name"`
	Kind     int       `sql:"kind"`
	Data     []byte    `sql:"data"`
	Modified time.Time `sql:"modified" update:"false"`
	astring  string
	anint    int
}

const createdb = `
create table if not exists structs (
id integer primary key,
name text,
kind integer,
data blob,
modified timestamp,
astring string,
anint integer
);`

const inserted = `
insert into paultest (name) values('Ringo');
`

func try() {
	dbu, err := rqlobj.NewRqlite(URL)
	if err != nil {
		panic(err)
	}
	query := createdb
	_, err = dbu.Write(query)
	if err != nil {
		fmt.Println("OOPSIE:\n", query, "\n")
		panic(err)
	}
	self := &testStruct{
		Name: "Bobby",
		Kind: 23,
		Data: []byte("what ev er"),
	}
	if err := dbu.Add(self); err != nil {
		panic(err)
	}
	fmt.Println("SAVED!")

}
