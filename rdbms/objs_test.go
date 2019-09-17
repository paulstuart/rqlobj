package main

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/paulstuart/rqlobj"
)

func TestCreate(t *testing.T) {
	var trace io.Writer
	if debug {
		trace = os.Stdout
	}
	dbu, err := rqlobj.NewRqlite(url, logger, trace)
	if err != nil {
		t.Fatalf("URL:%s err:%v", url, err)
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
	dbu, err := rqlobj.NewRqlite(url, logger, trace)
	if err != nil {
		t.Fatalf("URL:%s err:%v", url, err)
	}
	var list _testStruct
	if err := dbu.ListQuery(&list, "limit 5"); err != nil {
		t.Fatal(err)
	}
	for i, v := range list {
		t.Logf("%d: %+v\n", i, v)
	}
}
