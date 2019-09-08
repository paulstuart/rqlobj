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

func fieldList(list ...interface{}) string {
	var buf strings.Builder
	for i, item := range list {
		if i > 0 {
			buf.WriteString(", ")
		}
		switch item := item.(type) {
		case nil:
			fmt.Printf("item: %d is nil\n", i)
			buf.WriteString("null")
		case string:
			// escape any "'" by repeating them
			item = singleQuote.ReplaceAllString(item, "''")
			buf.WriteString("'")
			buf.WriteString(item)
			buf.WriteString("'")
		case []byte:
			// TODO: should []byte be base64 encoded?
			buf.WriteString("'")
			buf.WriteString(string(item))
			buf.WriteString("'")
		case time.Time:
			//fmt.Printf("FIELD:%d %v\n", i, item)
			if item.IsZero() {
				buf.WriteString("null")
			} else {
				buf.WriteString(fmt.Sprint(item.Unix()))
			}
		default:
			//fmt.Println("DEFAULT VALUE:", item)
			buf.WriteString(fmt.Sprint(item))
		}
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

func insertQuery(o DBObject) string {
	/*
		p := fields(o.InsertValues())
		return fmt.Sprintf("insert into %s (%s) values(%s)", o.TableName(), insertFields(o), p)
	*/
	values := o.InsertValues()
	fields := strings.Split(o.InsertFields(), ",")
	for i, p := range fields {
		if p == o.KeyField() {
			fields = append(fields[:i], fields[i+1:]...)
			values = append(values[:i], values[i+1:]...)
		} else if t, ok := values[i].(time.Time); ok && t.IsZero() {
			fields = append(fields[:i], fields[i+1:]...)
			values = append(values[:i], values[i+1:]...)
		}
	}
	const text = "insert into %s (%s) values(%s)"
	return fmt.Sprintf(text, o.TableName(), strings.Join(fields, ","), fieldList(values))
}

func replaceQuery(o DBObject) string {
	p := fieldList(o.InsertValues())
	return fmt.Sprintf("replace into %s (%s) values(%s)", o.TableName(), insertFields(o), p)
}

func updateQuery(o DBObject) string {
	//list := strings.Split(o.InsertFields(), ",")
	return fmt.Sprintf("update %s set %s where %s=%d", o.TableName(), setParams(insertFields(o)), o.KeyField(), o.Key())
}

func deleteQuery(o DBObject) string {
	// TODO: need to support non-int, multi-column keys
	return fmt.Sprintf("delete from %s where %s=%d", o.TableName(), o.KeyField(), o.Key())
}

func upsertQuery(o DBObject) string {
	/*
		var fields []string
		var values []interface{}
	*/
	values := o.InsertValues()
	fields := strings.Split(o.InsertFields(), ",")
	/*
		fmt.Printf("VALUES:")
		for i, f := range fields {
			fmt.Printf(" %s:%v", f, values[i])
		}
		fmt.Println("")
	*/
	//fmt.Printf("EQUAL %d => %d\n", len(fields), len(values))
	for i := 0; i < len(fields); i++ {
		p := fields[i]
		//fmt.Printf("====> I:%d FIELD:%s LIST (%d/%d):%v\n", i, p, len(fields), len(values), fields)
		/**
		drop := func() {
			fmt.Printf("TO:%d FROM:%d LIST:%v\n", i, i+1, fields)
			fields = append(fields[:i], fields[i+1:]...)
			values = append(values[:i], values[i+1:]...)
			i--
		}
		if p == o.KeyField() {
			fmt.Println("dropping key field:", p)
			drop()
		} else if t, ok := values[i].(time.Time); ok && t.IsZero() {
			fmt.Println("dropping unset time field:", p)
			drop()
		}
		**/
		/**/
		if p == o.KeyField() {
		} else if t, ok := values[i].(time.Time); ok && !t.IsZero() {
		} else {
			continue
		}
		fmt.Printf("DROP:%q TO:%d FROM:%d LIST:%v\n", p, i, i+1, fields)
		fields = append(fields[:i], fields[i+1:]...)
		values = append(values[:i], values[i+1:]...)
		i--
		/**/
	}
	/*
		fmt.Printf("NEW VALUES:")
		for i, f := range fields {
			fmt.Printf(" %s:%v", f, values[i])
		}
		fmt.Println("")
	*/
	const text = "insert into %s (%s) values(%s) on conflict(%s) do nothing"
	return fmt.Sprintf(text, o.TableName(), strings.Join(fields, ","), fieldList(values...), o.KeyField())
	//return fmt.Sprintf(text, o.TableName(), insertFields(o), fieldList(o.InsertValues()...), o.KeyField())
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

// Replace will replace an existing object in datastore
func (db DBU) Replace(o DBObject) error {
	results, err := db.dbs.Write([]string{replaceQuery(o)})
	if err != nil {
		o.SetID(results[0].LastInsertID)
	}
	return err
}

// Save modified object in datastore
func (db DBU) Save(o DBObject) error {
	query := upsertQuery(o)
	results, err := db.dbs.Write([]string{query})
	for _, result := range results {
		if result.Err != nil {
			// assuming that if there's an error here,
			// then the err value of the Write is non-nil
			log.Println("SAVE ERR:", result.Err)
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
	db.Debugf(deleteQuery(o), o.Key())
	_, err := db.dbs.Write([]string{deleteQuery(o)})
	return err
}

// DeleteByID object from datastore by id
func (db DBU) DeleteByID(o DBObject, id interface{}) error {
	db.Debugf(deleteQuery(o), id)
	_, err := db.dbs.Write([]string{deleteQuery(o)})
	return err
}

// List objects from datastore
func (db DBU) List(list DBList) error {
	return db.ListQuery(list, "")
}

// Find loads an object matching the given keys
func (db DBU) Find(o DBObject, keys map[string]interface{}) error {
	where := make([]string, 0, len(keys))
	what := make([]interface{}, 0, len(keys))
	for k, v := range keys {
		where = append(where, k+"=?")
		what = append(what, v)
	}
	query := fmt.Sprintf("select %s from %s where %s", o.SelectFields(), o.TableName(), strings.Join(where, " and "))
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
	if db.log != nil {
		db.log.Printf("QUERY:%s\n", query)
	}
	results, err := db.dbs.QueryOne(query)
	if err != nil {
		log.Println("error on query: " + query + " -- " + err.Error())
		return nil
	}
	return results.Scan(receivers)
}
