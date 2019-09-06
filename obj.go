package rqlobj

import (
	//"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	rqlite "github.com/rqlite/gorqlite"
	//rqlite "github.com/paulstuart/gorqlite"
)

var (
	// ErrNoKeyField is returned for tables without primary key identified
	ErrNoKeyField = errors.New("table has no key field")

	// ErrKeyMissing is returned when key value is not set
	ErrKeyMissing = errors.New("key is not set")

	ErrNilWritePointers = errors.New("nil record dest members")

	singleQuote = regexp.MustCompile("'")
)

/*
// Common Rows object between rqlite and /pkg/database/sql
// TODO: delete this?
type Common interface {
	Columns() []string
	Next() bool
	Scan(...interface{}) error
}
*/

// SQLDB is a common interface for opening an sql db
//type SQLDB func(string) (*sql.DB, error)

// SetHandler returns a slice of value pointer interfaces
// If there are no values to set it returns a nil instead
//type SetHandler func() []interface{}

// fragment to rethink code structure
/*
func commonQuery(rows Common, fn SetHandler) error {
	for rows.Next() {
		dest := fn()
		if dest == nil {
			return ErrNilWritePointers
		}
		if err := rows.Scan(dest...); err != nil {
			return err
		}
	}
	return nil
}
*/
type Fake struct {
	Number int       `sql:"numb"`
	TS     time.Time `sql:"ts"`
}

// DBU is a DatabaseUnit
type DBU struct {
	dbs *rqlite.Connection
	log *log.Logger
}

func (d DBU) Write(query ...string) ([]rqlite.WriteResult, error) {
	return d.dbs.Write(query)
}

// SetLogger sets the logger for the db
func (d DBU) SetLogger(logger *log.Logger) {
	d.log = logger
}

func (d DBU) debugf(msg string, args ...interface{}) {
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
	ModifiedBy(int64, time.Time)
}

func fields(list ...interface{}) string {
	var buf strings.Builder
	for i, item := range list {
		if i > 0 {
			buf.WriteString(", ")
		}
		switch item := item.(type) {
		case string:
			item = singleQuote.ReplaceAllString(item, "''")
			buf.WriteString("'")
			buf.WriteString(item)
			buf.WriteString("'")
		case []byte:
			buf.WriteString("'")
			buf.WriteString(string(item))
			buf.WriteString("'")
		case time.Time:
			buf.WriteString(fmt.Sprint(item.Unix()))
		default:
			buf.WriteString(fmt.Sprint(item))
		}
	}
	return buf.String()
}

/*
// renderedFields is because rqlite doesn't support bind parameters
func renderedFields(values ...interface{}) string {
	var buf strings.Builder
	for i, value := range values {
		if i > 0 {
			buf.WriteString(", ")
		}
		switch value := value.(type) {
		case string:
			value = singleQuote.ReplaceAllString(value, "''")
			buf.WriteString("'")
			buf.WriteString(fmt.Sprint(value))
			buf.WriteString("'")
		case []byte:
			buf.WriteString("'")
			buf.WriteString(string(value))
			buf.WriteString("'")
		default:
			buf.WriteString(fmt.Sprint(value))
		}
	}
	return buf.String()
}
*/

func insertFields(o DBObject) string {
	list := strings.Split(o.InsertFields(), ",")
	keep := make([]string, 0, len(list))
	for _, p := range list {
		if p != o.KeyField() {
			keep = append(keep, p)
		}
	}
	return strings.Join(keep, ",")
}

func setParams(params string) string {
	list := strings.Split(params, ",")
	for i, p := range list {
		list[i] = fmt.Sprintf("%s=?", p)
	}
	return strings.Join(list, ",")
}

func insertQuery(o DBObject) string {
	p := fields(o.InsertValues())
	return fmt.Sprintf("insert into %s (%s) values(%s)", o.TableName(), insertFields(o), p)
}

func replaceQuery(o DBObject) string {
	p := fields(o.InsertValues())
	return fmt.Sprintf("replace into %s (%s) values(%s)", o.TableName(), insertFields(o), p)
}

func updateQuery(o DBObject) string {
	//list := strings.Split(o.InsertFields(), ",")
	return fmt.Sprintf("update %s set %s where %s=%s", o.TableName(), setParams(insertFields(o)), o.KeyField())
}

func deleteQuery(o DBObject) string {
	//key := o.Key()
	// TODO: need to support non-int, multi-column keys
	return fmt.Sprintf("delete from %s where %s=%d", o.TableName(), o.KeyField(), o.Key())
}

func upsertQuery(o DBObject) string {
	//args := o.InsertValues()
	const text = "insert into %s (%s) values(%s) on conflict(%s) do nothing"
	return fmt.Sprintf(text, o.TableName(), insertFields(o), fields(o.InsertValues()...), o.KeyField())
}

// Add new object to datastore
func (db DBU) Add(o DBObject) error {
	query := upsertQuery(o)
	fmt.Println("ADD QUERY:", query)
	results, err := db.dbs.Write([]string{query})
	for _, result := range results {
		fmt.Println("ADD ERR:", result.Err)
	}
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
	fmt.Println("SAVE:", updateQuery(o))
	_, err := db.dbs.Write([]string{updateQuery(o)})
	return err
}

// Delete object from datastore
func (db DBU) Delete(o DBObject) error {
	db.debugf(deleteQuery(o), o.Key())
	_, err := db.dbs.Write([]string{deleteQuery(o)})
	return err
}

// DeleteByID object from datastore by id
func (db DBU) DeleteByID(o DBObject, id interface{}) error {
	db.debugf(deleteQuery(o), id)
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

type Scanner interface {
	Scan(...interface{}) error
	//Next() bool
}

// DBList is the interface for a list of db objects
type DBList interface {
	/*
		QueryString(extra string) string
		Receivers() []interface{}
	*/
	SQLGet(extra string) string
	SQLResults(Scanner) error
}

// ListQuery updates a list of objects
// TODO: handle args/vs no args for rqlite
func (db DBU) ListQuery(list DBList, extra string) error {
	/*
		fn := func() []interface{} {
			return list.Receivers()
		}
		query := list.QueryString(extra)
		return db.dbs.Query(fn, query)
	*/
	query := list.SQLGet("")
	results, err := db.dbs.Query([]string{query})
	if err != nil {
		return err
	}
	for _, result := range results {
		for result.Next() {
			if err := list.SQLResults(&result); err != nil {
				return err
			}
		}
	}
	return nil
}

// get is the low level db wrapper
func (db DBU) get(members []interface{}, query string) error {
	if db.log != nil {
		db.log.Printf("QUERY:%s\n", query)
	}
	results, err := db.dbs.QueryOne(query)
	if err != nil {
		log.Println("error on query: " + query + " -- " + err.Error())
		return nil
	}
	return results.Scan(members)
}
