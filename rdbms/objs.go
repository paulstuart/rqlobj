package main

import (
	"time"
)

type testStruct struct {
	ID        int64     `sql:"id" key:"true" table:"structs"`
	Name      string    `sql:"name"`
	Kind      int64     `sql:"kind"`
	Data      string    `sql:"data"`
	Timestamp time.Time `sql:"ts"`
	When      time.Time `sql:"ts2"`
	astring   string
	anint     int
	/*
		Modified  time.Time `sql:"modified" update:"false"`
		Kind      int       `sql:"kind"`
		Data      []byte    `sql:"data"`
	*/
}
