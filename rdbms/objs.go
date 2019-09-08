package main

import (
	"time"
)

type testStruct struct {
	ID   int64  `sql:"id" key:"true" table:"structs"`
	Name string `sql:"name"`
	//Kind int    `sql:"kind"`
	Kind int64 `sql:"kind"`
	//Data     []byte    `sql:"data"`
	Data      string    `sql:"data"`
	When      time.Time `sql:"ts2"`
	Timestamp time.Time `sql:"ts"`
	//Modified time.Time `sql:"modified" update:"false"`
	astring string
	anint   int
}
