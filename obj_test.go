package rqlobj

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	trace bool
)

const tableName = "test_structs"

type testStruct struct {
	// struct tag must be single string,
	// all other table name refs should refer to tableName
	ID       int64     `sql:"id" key:"true" table:"test_structs"`
	Name     string    `sql:"name"`
	Kind     int       `sql:"kind"`
	Data     string    `sql:"data"`
	Modified time.Time `sql:"modified" update:"false"`
	astring  string
	anint    int
}

func (s *testStruct) Names() []string {
	return []string{
		"ID",
		"Name",
		"Kind",
		"Data",
		"Modified",
	}
}

func (s *testStruct) TableName() string {
	return tableName //"structs"
}

func (s *testStruct) KeyField() string {
	return "id"
}

func (s *testStruct) KeyName() string {
	return "ID"
}

func (s *testStruct) InsertFields() string {
	return "name,kind,data"
}

func (s *testStruct) SelectFields() string {
	return "id,name,kind,data,modified"
}

func (s *testStruct) UpdateValues() []interface{} {
	return []interface{}{s.Name, s.Kind, s.Data, s.ID}
}

func (s *testStruct) MemberPointers() []interface{} {
	return []interface{}{&s.ID, &s.Name, &s.Kind, &s.Data, &s.Modified}
}

func (s *testStruct) InsertValues() []interface{} {
	return []interface{}{s.Name, s.Kind, s.Data}
}

func (s *testStruct) SetID(id int64) {
	s.ID = id
}

func (s *testStruct) Key() int64 {
	return s.ID
}

func (s *testStruct) ModifiedBy(u int64, t time.Time) {
	s.Modified = t
}

type _testStruct []testStruct

func (s *_testStruct) SQLGet(extra string) string {
	// TODO: any way to derive from _testStruct?
	const fields = "id,name,kind,data,modified"
	query := fmt.Sprintf("select %s from %s", fields, tableName)
	if extra != "" {
		if !strings.HasPrefix(strings.ToLower(extra), "limit") {
			query += " where " + extra
		} else {
			query += " " + extra
		}
	}
	return query
}

func (s *_testStruct) SQLResults(fn func(...interface{}) error) error {
	var add testStruct
	if err := fn((&add).MemberPointers()...); err != nil {
		return err
	}
	*s = append(*s, add)
	return nil
}

const queryCreate = `create table if not exists ` + tableName + ` (
    id integer not null primary key,
    name text,
    kind int,
    data blob,
    modified DATETIME DEFAULT CURRENT_TIMESTAMP
);`

type testMap map[int64]testStruct

func structDb(t *testing.T) DBU {
	var out, w io.Writer
	if trace {
		w = os.Stdout
	}
	if testing.Verbose() {
		out = os.Stdout
	}
	dbs, err := NewRqlite("http://localhost:4001", out, w)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dbs.dbs.Write(canned()); err != nil {
		t.Fatal(err)
	}
	return dbs
}

func TestMain(m *testing.M) {
	flag.BoolVar(&trace, "trace", false, "trace rqlite calls")
	flag.Parse()
	os.Exit(m.Run())
}

func TestFindBy(t *testing.T) {
	db := structDb(t)
	s := &testStruct{
		Name: "Waldo",
		Kind: 1982,
		Data: test_data,
	}
	if err := db.Add(s); err != nil {
		t.Fatal(err)
	}
	f := &testStruct{
		ID: s.ID,
	}
	if err := db.FindBy(f, s.KeyField(), f.ID); err != nil {
		t.Error(err)
	}
	t.Log("BY ID", f)
	u := testStruct{}
	if err := db.FindBy(&u, "name", s.Name); err != nil {
		t.Error(err)
	}
	t.Log("BY NAME", u)
	if err := db.Delete(f); err != nil {
		t.Error(err)
	}
}

func TestSelf(t *testing.T) {
	db := structDb(t)
	s := testStruct{ID: 1}
	if err := db.FindSelf(&s); err != nil {
		t.Error(err)
	}
	t.Log("BY SELF", s)
}

var test_data = "lorem ipsum"

func TestDBObject(t *testing.T) {
	db := structDb(t)
	s := &testStruct{
		Name: "Grammatic, Bro",
		Kind: 2001,
		Data: test_data,
	}
	if err := db.Add(s); err != nil {
		t.Fatal(err)
	}
	s.Kind = 2015
	s.Name = "Void droid"
	if err := db.Update(s); err != nil {
		t.Fatal(err)
	}
	z := testStruct{}
	m := map[string]interface{}{"kind": 2015}
	if err := db.Find(&z, m); err != nil {
		t.Fatal(err)
	}

	if err := db.Delete(s); err != nil {
		t.Fatal(err)
	}
}

func testDBU(t *testing.T) *sql.DB {
	return nil
}

func canned() []string {
	const queryInsert = "insert into " + tableName + " (name, kind, data) values(%s)"
	var queries []string
	prep := func(query string, args ...interface{}) {
		if len(args) == 0 {
			queries = append(queries, query)
		} else {
			query = fmt.Sprintf(query, fieldList(args...))
			queries = append(queries, query)
		}
	}
	prep(queryCreate)
	prep(queryInsert, "abc", 23, "what ev er")
	prep(queryInsert, "def", 69, "m'kay")
	prep(queryInsert, "ghi", 42, "meaning of life")
	prep(queryInsert, "jkl", 2, "of a kind")
	prep(queryInsert, "mno", 2, "of a drag")
	prep(queryInsert, "pqr", 2, "of a sort")
	return queries
}

func dump(t *testing.T, db *sql.DB, query string, args ...interface{}) {
	rows, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	dest := make([]interface{}, len(args))
	for i, f := range args {
		dest[i] = &f
	}
	for rows.Next() {
		rows.Scan(dest...)
		t.Log(args...)
	}
	rows.Close()
}

func TestListQuery(t *testing.T) {
	db := structDb(t)
	list := new(_testStruct)
	if err := db.ListQuery(list, "limit 5"); err != nil {
		t.Fatal(err)
	}
	for _, item := range *list {
		t.Logf("ITEM:  %+v\n", item)
	}
	list = new(_testStruct)
	db.ListQuery(list, "(id % 10) = 0 limit 5")
	for _, item := range *list {
		t.Logf("TENS:  %+v\n", item)
	}
}

func TestDelete(t *testing.T) {
	db := structDb(t)
	s := &testStruct{
		Name: "Lost Record",
		Kind: 2001,
	}
	if err := db.Add(s); err != nil {
		t.Fatal(err)
	}
	t.Logf("New ID:%d", s.ID)
	if err := db.Delete(s); err != nil {
		t.Fatal(err)
	}
	s.ID = -1
	if err := db.Delete(s); err != nil {
		t.Logf("expected delete error: %v", err)
	} else {
		t.Fatal("expected error but got none")
	}
}
