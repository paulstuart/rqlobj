// Package rqlobj provides basic object relational mapping against an sql database
//
// Originally developed against SQLite using runtime reflection to generate queries.
//
// The next version incorporated SQL fragment generation that exposed the struct tag
// data and allowed creating queries without reflection.
//
// This current version extends to multi-key support and an ostensibly trimmer API
package rqlobj

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/paulstuart/gorqlite"
	"github.com/pkg/errors"
)

var (
	// ErrNoKeyField is returned for tables without primary key identified
	ErrNoKeyField = errors.New("table has no key field")

	// ErrKeyMissing is returned when key value is not set
	ErrKeyMissing = errors.New("key is not set")

	singleQuote = regexp.MustCompile("'")
)

// DBU is a database handler
type DBU struct {
	dbs   *gorqlite.Connection
	debug bool
	_log  *log.Logger
}

// Write will process a batch of queries and return a batch of results
func (db DBU) Write(queries ...string) ([]gorqlite.WriteResult, error) {
	if db.debug {
		for _, query := range queries {
			db.debugf("Write: %s\n", query)
		}
	}
	return db.dbs.Write(queries)
}

// SetLogger sets the logger for the db
func (db DBU) SetLogger(w io.Writer) {
	flags := log.Ldate | log.Lmicroseconds | log.Lshortfile
	db._log = log.New(w, "", flags)
}

// debugf sends to common log
func (db DBU) debugf(msg string, args ...interface{}) {
	if db._log != nil {
		db._log.Printf(msg, args...)
	}
}

/*
CREATE
READ
UPDATE
DELETE
*/
// DXo object
type DXo interface {
	// return an unqualified query that selects all fields
	SelectQuery() string

	// return the 'where' string to retrieve an object by it's primary key(s)
	SelectWhere() string

	// return a list of pointers for all fields
	SelectReceivers() []interface{}
}

// DBObject interface provides methods for object storage
// in an sql database.  The functions are generated for each
// relevant struct type that are annotated accordingly
type DBObject interface {
	// TableName is the name of the sql table
	TableName() string

	// Primary returns an int64 and a bool
	// representing the int64 primary key, and 'true'
	// if it is valid (0 is valid for uninitialized)
	Primary() (int64, bool)

	// SetPrimary updates the primary key
	SetPrimary(int64)

	// KeyNames are the struct names of the
	// primary id fields
	KeyNames() []string

	// KeyValues returns the value(s) of the object key(s)
	KeyValues() []interface{}

	// KeyFields are the names of the table fields
	// comprising the primary id
	KeyFields() []string

	// Names returns the struct element names
	Names() []string

	// SelectFields returns the comma separated
	// list of fields to be selected
	SelectFields() string

	// InsertFields returns the comma separated
	// list of fields to be inserted
	InsertFields() string

	// InsertValues returns the values of the object to be inserted
	InsertValues() []interface{}

	// UpdateValues returns the values of the object to be updated
	UpdateValues() []interface{}

	// Receivers  returns a slice of pointers to values
	// for the db Scan function
	Receivers() []interface{}
}

// prepare item as sql query value
func formatted(item interface{}) string {
	switch item := item.(type) {
	case nil:
		return "null"
	case string:
		// escape any "'" by repeating them
		return "'" + singleQuote.ReplaceAllString(item, "''") + "'"
	case []byte:
		return "'" + singleQuote.ReplaceAllString(string(item), "''") + "'"
	case time.Time:
		if item.IsZero() {
			return "null"
		}
		return fmt.Sprint(item.Unix())
	}
	return fmt.Sprint(item)
}

func fieldList(list ...interface{}) string {
	var buf strings.Builder
	for i, item := range list {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(formatted(item))
	}
	return buf.String()
}

func insertFields(o DBObject) string {
	list := strings.Split(o.InsertFields(), ",")
	keys := o.KeyFields()
	omit := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		omit[k] = struct{}{}
	}
	for i, p := range list {
		if _, ok := omit[p]; ok {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return strings.Join(list, ",")
}

func setParams(params string) string {
	list := strings.Split(params, ",")
	for i, p := range list {
		list[i] = fmt.Sprintf("%s=?", p)
	}
	return strings.Join(list, ",")
}

func replaceQuery(o DBObject) string {
	p := fieldList(o.InsertValues())
	return fmt.Sprintf("replace into %s (%s) values(%s)", o.TableName(), insertFields(o), p)
}

func updateQuery(o DBObject) string {
	panic("fix the 'set' part of the update statement")
	if id, ok := o.Primary(); ok {
		return fmt.Sprintf("update %s set %s where %s=%d", o.TableName(), setParams(insertFields(o)), o.KeyFields()[0], id)
	}
	values := o.KeyValues()
	var where strings.Builder
	for i, key := range o.KeyFields() {
		if i > 0 {
			where.WriteString(" and ")
		}
		where.WriteString(key)
		where.WriteString("=")
		switch val := values[i].(type) {
		case string:
			where.WriteByte('\'')
			where.WriteString(val)
			where.WriteByte('\'')
		default:
			where.WriteString(fmt.Sprint(val))
		}
	}

	return fmt.Sprintf("update %s set %s where %s", o.TableName(), setParams(insertFields(o)), where.String())
}

func deleteQuery(o DBObject, key int64) string {
	// TODO: need to support non-int, multi-column keys
	if key == 0 {
		return fmt.Sprintf("delete from %s;", o.TableName())
	}
	return fmt.Sprintf("delete from %s where %s=%v;", o.TableName(), o.KeyFields(), key)
}

func within(s string, list []string) bool {
	for _, item := range list {
		if s == item {
			return true
		}
	}
	return false
}

func join(list []string) string {
	return strings.Join(list, ",")
}

func upsertQuery(o DBObject) string {
	values := o.InsertValues()
	fields := strings.Split(o.InsertFields(), ",")
	// do not include fields with unset time -- it's effectively null
	for i := 0; i < len(fields); i++ {
		//fmt.Printf("%d/%d %T:%v\n", i+1, len(values), values[i], values[i])
		p := fields[i]
		//if p == o.KeyFields() {
		if within(p, o.KeyFields()) {
			// nada, just to skip the continue
		} else if t, ok := values[i].(time.Time); ok && !t.IsZero() {
			// nada, just to skip the continue
		} else {
			continue
		}
		i--
		fields = append(fields[:i], fields[i+1:]...)
		values = append(values[:i], values[i+1:]...)
	}
	const text = "INSERT into %s (%s) values(%s) on conflict(%s) do nothing"
	return fmt.Sprintf(text, o.TableName(), join(fields), fieldList(values...), join(o.KeyFields()))
}

// Add new object to datastore
func (db DBU) Add(o DBObject) error {
	query := upsertQuery(o)
	results, err := db.Write(query)
	if err != nil {
		for _, result := range results {
			fmt.Println("RES ERR:", result.Err)
		}
		return err
	}
	if len(results) > 0 {
		o.SetPrimary(results[0].LastInsertID)
	}
	return nil
}

// Update saves a modified object in the datastore
func (db DBU) Update(o DBObject) error {
	query := upsertQuery(o)
	results, err := db.Write(query)
	for _, result := range results {
		if result.Err != nil {
			// assuming that if there's an error here,
			// then the err value of the Write is non-nil
			db.debugf("save error: %+v\n", result.Err)
		}
	}
	if err != nil {
		return err
	}
	if len(results) > 0 {
		o.SetPrimary(results[0].LastInsertID)
	}
	return nil
}

// Delete object from datastore
func (db DBU) Delete(o DBObject) error {
	if id, ok := o.Primary(); ok {
		return db.DeleteByID(o, id)
	}
	return nil
}

// DeleteByID object from datastore by id
func (db DBU) DeleteByID(o DBObject, id int64) error {
	query := deleteQuery(o, id)
	db.debugf(query)
	results, err := db.Write(query)
	if err != nil {
		return err
	}
	for _, result := range results {
		if result.RowsAffected == 0 {
			return errors.New("no rows deleted")
		}
	}
	return nil
}

// DeleteAll deletes all objects of that type from the datastore
func (db DBU) DeleteAll(o DBObject) error {
	return db.DeleteByID(o, 0)
}

// Load loads an object matching the given keys
func (db DBU) Load(o DBObject, keys map[string]interface{}) error {
	where := make([]string, 0, len(keys))
	for k, v := range keys {
		where = append(where, fmt.Sprintf("%s=%v", k, v))
	}
	const text = "select %s from %s where %s"
	query := fmt.Sprintf(text, o.SelectFields(), o.TableName(), strings.Join(where, " and "))
	return db.get(o.Receivers(), query)
}

// LoadBy loads an  object matching the given key/value
func (db DBU) LoadBy(o DBObject, key string, value interface{}) error {
	var text, query string
	switch value := value.(type) {
	case string:
		text = "select %s from %s where %s='%s'"
		query = fmt.Sprintf(text, o.SelectFields(), o.TableName(), key, value)
	case int, int64, uint, uint64:
		text = "select %s from %s where %s=%d"
		query = fmt.Sprintf(text, o.SelectFields(), o.TableName(), key, value)
	default:
		text = "select %s from %s where %s=%v"
		query = fmt.Sprintf(text, o.SelectFields(), o.TableName(), key, value)
	}
	return db.get(o.Receivers(), query)
}

// LoadByID loads an object based on a given int64 primary ID
func (db DBU) LoadByID(o DBObject, id int64) error {
	const text = "select %s from %s where %s=%d"
	if id, ok := o.Primary(); ok {
		return db.LoadBy(o, o.KeyFields()[0], id)
	}
	return fmt.Errorf("does not have an int primary id")

}

// LoadSelf loads an object based on it's current ID
func (db DBU) LoadSelf(o DBObject) error {
	if id, ok := o.Primary(); ok {
		return db.LoadBy(o, o.KeyFields()[0], id)
	}
	if len(o.KeyFields()) == 0 {
		return ErrNoKeyField
	}
	keys := o.KeyFields()
	if len(keys) == 1 {
		return db.LoadBy(o, keys[0], o.KeyValues()[0])
	}
	if len(keys) == 0 {
		return ErrKeyMissing
	}
	values := o.KeyValues()
	m := make(map[string]interface{}, len(keys))
	for i, key := range keys {
		m[key] = values[i]
	}

	return db.Load(o, m)
}

// DBList is the interface for a list of db objects
type DBList interface {
	// SQLGet is the query string to retrieve the list
	SQLGet(extra string) string

	// SQLResults takes a Scan function
	SQLResults(func(...interface{}) error) error
	//SQLResults() []interface{}
}

// typeinfo returns a string with interface info
func typeinfo(list ...interface{}) string {
	var buf strings.Builder
	for i, item := range list {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf("%d:%T:%v", i, item, item))
	}
	return buf.String()
}

// List objects from datastore
func (db DBU) List(list DBList) error {
	return db.ListQuery(list, "")
}

// ListQuery updates a list of objects
func (db DBU) ListQuery(list DBList, where string) error {
	query := list.SQLGet(where)
	results, err := db.dbs.Query([]string{query})
	if err != nil {
		for _, res := range results {
			fmt.Println("LQ ERR:", res.Err)
		}
		return err
	}
	for _, result := range results {
		for result.Next() {
			fn := func(ptrs ...interface{}) error {
				if err := result.Scan(ptrs...); err != nil {
					return fmt.Errorf("%w: with ptrs: %s", err, typeinfo(ptrs...))
				}
				return err
			}
			if err := list.SQLResults(fn); err != nil {
				db.debugf("scan error: %v\n", err)
				return err
			}
		}
	}
	return nil
}

// get is the low level db wrapper
func (db DBU) get(receivers []interface{}, query string) error {
	db.debugf("get query:%s\n", query)
	result, err := db.dbs.QueryOne(query)
	if err != nil {
		db.debugf("error on get query: %q :: %v\n", query, err)
		return err
	}
	if result.Next() {
		return result.Scan(receivers...)
	}
	return nil
}

// NewRqlite returns a DBU connected to a rqlite cluster
func NewRqlite(host string, logger, trace io.Writer) (DBU, error) {
	conn, err := gorqlite.Open(host)
	if logger == nil {
		logger = ioutil.Discard
	}
	dbu := DBU{_log: log.New(logger, "", 0)}
	if err == nil {
		if trace != nil {
			gorqlite.TraceOn(trace)
			dbu.debug = true
		}
		dbu.dbs = &conn
	}
	return dbu, err
}
