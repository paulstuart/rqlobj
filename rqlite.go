package rqlobj

import (
	"fmt"

	//rqlite "github.com/rqlite/gorqlite"
	rqlite "github.com/paulstuart/gorqlite"
)

type rqliteWrapper struct {
	conn *rqlite.Connection
}

func (s rqliteWrapper) Query(fn SetHandler, query string, args ...interface{}) error {
	// TODO: include args!
	// TODO: build query buffer to batch
	queries := []string{query}
	results, err := s.conn.Query(queries)
	if err != nil {
		return err
	}
	for _, result := range results {
		for result.Next() {
			dest := fn()
			if dest == nil {
				return ErrNilWritePointers
			}
			if err = result.Scan(dest...); err != nil {
				return err
			}
		}
	}
	return nil
}

/*
type WriteResult struct {
    Err          error // don't trust the rest if this isn't nil
    Timing       float64
    RowsAffected int64 // affected by the change
    LastInsertID int64 // if relevant, otherwise zero value
    // contains filtered or unexported fields
}
*/
// TODO: include args in query
func (s rqliteWrapper) Exec(query string, args ...interface{}) (rowsAffected, lastInsertID int64, err error) {
	results, err := s.conn.Write([]string{query})
	for _, result := range results {
		return result.RowsAffected, result.LastInsertID, result.Err
	}

	return 0, 0, nil
}

//func NewRqlite(addr string) (*rqliteWrapper, error) {
func NewRqlite(addr string) (*DBU, error) {
	//	panic: interface conversion: interface {} is string, not map[string]interface {}
	fmt.Println("open addr:", addr)
	r, err := rqlite.Open(addr)
	fmt.Println("opened addr:", addr)
	return &DBU{
		dbs: &r,
	}, err
}
