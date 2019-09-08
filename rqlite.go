package rqlobj

import (
	"io"
	"io/ioutil"
	"log"

	"github.com/rqlite/gorqlite"
)

func NewRqlite(host string, logger, trace io.Writer) (DBU, error) {
	conn, err := gorqlite.Open(host)
	if logger == nil {
		logger = ioutil.Discard
	}
	dbu := DBU{log: log.New(logger, "", 0)}
	if err == nil {
		if trace != nil {
			gorqlite.TraceOn(trace)
		}
		dbu.dbs = &conn
	}
	return dbu, err
}
