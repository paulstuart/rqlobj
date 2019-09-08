package rqlobj

import (
	"io"

	"github.com/rqlite/gorqlite"
)

func NewRqlite(host string, trace io.Writer) (DBU, error) {
	conn, err := gorqlite.Open(host)
	var dbu DBU
	if err == nil {
		if trace != nil {
			gorqlite.TraceOn(trace)
		}
		dbu.dbs = &conn
	}
	return dbu, err
}
