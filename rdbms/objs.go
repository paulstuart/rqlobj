package main

import (
	"time"
)

type testStruct struct {
	ID        int64     `sql:"id,key" table:"rdbms_structs"`
	Name      string    `sql:"name"`
	Kind      int64     `sql:"kind"`
	Data      string    `sql:"data"`
	Timestamp time.Time `sql:"ts"`
	When      time.Time `sql:"ts2"`
	astring   string
	anint     int
}

// for testing permutations of saving timestamps in sqlite
type testDates struct {
	ID        int64     `sql:"id,key" table:"rdbms_dates"`
	Name      string    `sql:"name"`
	Kind      int64     `sql:"kind"`
	Data      string    `sql:"data"`
	Timestamp time.Time `sql:"ts"`
	When      time.Time `sql:"ts2"`
	TS3       time.Time `sql:"ts3"`
	TS4       time.Time `sql:"ts4"`
	TS5       time.Time `sql:"ts5"`
}
