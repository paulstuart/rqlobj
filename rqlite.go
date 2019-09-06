package rqlobj

import (
	rqlite "github.com/rqlite/gorqlite"
)

func NewRqlite(host string) (DBU, error) {
	conn, err := rqlite.Open(host)
	var dbu DBU
	if err == nil {
		dbu.dbs = &conn
	}
	return dbu, err
}
