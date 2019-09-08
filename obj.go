package rqlobj

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	rqlite "github.com/rqlite/gorqlite"
)

var (
	// ErrNoKeyField is returned for tables without primary key identified
	ErrNoKeyField = errors.New("table has no key field")

	// ErrKeyMissing is returned when key value is not set
	ErrKeyMissing = errors.New("key is not set")

	ErrNilWritePointers = errors.New("nil record dest members")

	singleQuote = regexp.MustCompile("'")
)

// DBU is a Database abstraction Unit
type DBU struct {
	dbs *rqlite.Connection
	log *log.Logger
}

// Write will process a batch of queries and return a batch of results
func (d DBU) Write(query ...string) ([]rqlite.WriteResult, error) {
	return d.dbs.Write(query)
}

// SetLogger sets the logger for the db
func (d DBU) SetLogger(w io.Writer) {
	flags := log.Ldate | log.Lmicroseconds | log.Lshortfile
	d.log = log.New(w, "", flags)
}

// Debugf sends to common log
func (d DBU) Debugf(msg string, args ...interface{}) {
	if d.log != nil {
		d.log.Printf(msg, args...)
	}
}

// DBObject provides methods for object storage
// The functions are generated for each object
// annotated accordingly
type DBObject interface {
	// TableName is the name of the sql table
	TableName() string

	// KeyFields are the names of the table fields
	// comprising the primary id
	//KeyFields() []string
	KeyField() string

	// KeyNames are the struct names of the
	// primary id fields
	//KeyNames() []string
	KeyName() string

	// Names returns the struct element names
	Names() []string

	// SelectFields returns the comma separated
	// list of fields to be selected
	SelectFields() string

	// InsertFields returns the comma separated
	// list of fields to be selected
	InsertFields() string

	// Key returns the int64 id value of the object
	Key() int64

	// SetID updates the id of the object
	SetID(int64)

	// InsertValues returns the values of the object to be inserted
	InsertValues() []interface{}

	// InsertValues returns the values of the object to be updated
	UpdateValues() []interface{}

	// MemberPointers  returns a slice of pointers to values
	// for the db scan function
	MemberPointers() []interface{}

	// ModifiedBy returns the user id and timestamp of when the object was last modified
	//ModifiedBy(int64, time.Time)
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
	for i, p := range list {
		if p == o.KeyField() {
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
	return fmt.Sprintf("update %s set %s where %s=%d", o.TableName(), setParams(insertFields(o)), o.KeyField(), o.Key())
}

func deleteQuery(o DBObject, key int64) string {
	// TODO: need to support non-int, multi-column keys
	return fmt.Sprintf("delete from %s where %s=%v;", o.TableName(), o.KeyField(), key)
}

func upsertQuery(o DBObject) string {
	values := o.InsertValues()
	fields := strings.Split(o.InsertFields(), ",")
	// do not include fields with unset time -- it's effectively null
	for i := 0; i < len(fields); i++ {
		p := fields[i]
		if p == o.KeyField() {
		} else if t, ok := values[i].(time.Time); ok && !t.IsZero() {
		} else {
			continue
		}
		fields = append(fields[:i], fields[i+1:]...)
		values = append(values[:i], values[i+1:]...)
		i--
	}
	const text = "insert into %s (%s) values(%s) on conflict(%s) do nothing"
	return fmt.Sprintf(text, o.TableName(), strings.Join(fields, ","), fieldList(values...), o.KeyField())
}

// Add new object to datastore
func (db DBU) Add(o DBObject) error {
	query := upsertQuery(o)
	results, err := db.dbs.Write([]string{query})
	if err != nil {
		return err
	}
	if len(results) > 0 {
		o.SetID(results[0].LastInsertID)
	}
	return nil
}

// Save modified object in datastore
func (db DBU) Save(o DBObject) error {
	query := upsertQuery(o)
	results, err := db.dbs.Write([]string{query})
	for _, result := range results {
		if result.Err != nil {
			// assuming that if there's an error here,
			// then the err value of the Write is non-nil
			db.log.Println("SAVE ERR:", result.Err)
		}
	}
	if err != nil {
		return err
	}
	if len(results) > 0 {
		o.SetID(results[0].LastInsertID)
	}
	return nil
}

// Delete object from datastore
func (db DBU) Delete(o DBObject) error {
	query := deleteQuery(o, o.Key())
	db.Debugf(query)
	results, err := db.dbs.Write([]string{query})
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

// DeleteByID object from datastore by id
func (db DBU) DeleteByID(o DBObject, id interface{}) error {
	query := deleteQuery(o, id.(int64))
	db.Debugf(query)
	_, err := db.dbs.Write([]string{query})
	return err
}

// List objects from datastore
func (db DBU) List(list DBList) error {
	return db.ListQuery(list, "")
}

// Find loads an object matching the given keys
func (db DBU) Find(o DBObject, keys map[string]interface{}) error {
	where := make([]string, 0, len(keys))
	//what := make([]interface{}, 0, len(keys))
	for k, v := range keys {
		where = append(where, fmt.Sprintf("%s=%v", k, v))
	}
	const text = "select %s from %s where %s"
	query := fmt.Sprintf(text, o.SelectFields(), o.TableName(), strings.Join(where, " and "))
	return db.get(o.MemberPointers(), query)
}

// FindBy loads an  object matching the given key/value
func (db DBU) FindBy(o DBObject, key string, value interface{}) error {
	text := "select %s from %s where %s=%v"
	if _, ok := value.(string); ok {
		text = "select %s from %s where %s='%s'"
	}
	query := fmt.Sprintf(text, o.SelectFields(), o.TableName(), key, value)
	return db.get(o.MemberPointers(), query)
}

// FindByID loads an object based on a given ID
func (db DBU) FindByID(o DBObject, value interface{}) error {
	return db.FindBy(o, o.KeyField(), value)
}

// FindSelf loads an object based on it's current ID
func (db DBU) FindSelf(o DBObject) error {
	if len(o.KeyField()) == 0 {
		return ErrNoKeyField
	}
	if o.Key() == 0 {
		return ErrKeyMissing
	}
	return db.FindBy(o, o.KeyField(), o.Key())
}

// DBList is the interface for a list of db objects
type DBList interface {

	// SQLGet is the query string to retrieve the list
	SQLGet(extra string) string

	// SQLResults takes a Scan function
	SQLResults(func(...interface{}) error) error
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

// ListQuery updates a list of objects
// TODO: handle args/vs no args for rqlite
func (db DBU) ListQuery(list DBList, extra string) error {
	query := list.SQLGet("")
	results, err := db.dbs.Query([]string{query})
	if err != nil {
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
				return err
			}
		}
	}
	return nil
}

// get is the low level db wrapper
func (db DBU) get(receivers []interface{}, query string) error {
	db.Debugf("get query:%s\n", query)
	result, err := db.dbs.QueryOne(query)
	if err != nil {
		db.log.Println("error on get query: " + query + " -- " + err.Error())
		return err
	}
	if result.Next() {
		return result.Scan(receivers...)
	}
	return nil
}
