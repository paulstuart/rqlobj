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

func init() {
	// *** for now ***
	// make sure proxy is set up (rqlite is inside docker)
	os.Setenv("http_proxy", "http://localhost:8888/")
}

func (s *testStruct) equal(other *testStruct) error {
	if s.ID != other.ID {
		return fmt.Errorf("New ID: %d doesn't match orig: %d\n", other.ID, s.ID)
	}
	if s.Name != other.Name {
		return fmt.Errorf("New Name: %s doesn't match orig: %s\n", other.Name, s.Name)
	}
	if s.Kind != other.Kind {
		return fmt.Errorf("New Kind: %d doesn't match orig: %d\n", other.Kind, s.Kind)
	}
	if s.Data != other.Data {
		return fmt.Errorf("New Data: %s doesn't match orig: %s\n", other.Data, s.Data)
	}
	const within = time.Second * 2
	if s.Modified.UTC().Truncate(within) != other.Modified.UTC().Truncate(within) {
		return fmt.Errorf("New Modified: %s doesn't match orig: %s\n", other.Modified, s.Modified)
	}
	return nil
}

func (s *testStruct) SQLCreate() string {
	return `create table if not exists ` + tableName + ` (
    id integer not null primary key,
    name text,
    kind int,
    data blob,
    modified DEFAULT (datetime('now','utc'))
);`
}

// modified DATETIME DEFAULT CURRENT_TIMESTAMP

func (s *testStruct) Elements() []string {
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

func (s *testStruct) KeyFields() []string {
	return []string{"id"}
}

func (s *testStruct) KeyNames() []string {
	return []string{"ID"}
}

func (s *testStruct) KeyValues() []interface{} {
	return []interface{}{s.ID}
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

func (s *testStruct) Receivers() []interface{} {
	return []interface{}{&s.ID, &s.Name, &s.Kind, &s.Data, &s.Modified}
}

func (s *testStruct) InsertValues() []interface{} {
	return []interface{}{s.Name, s.Kind, s.Data}
}

func (s *testStruct) SetPrimary(id int64) {
	s.ID = id
}

func (s *testStruct) Primary() (int64, bool) {
	return s.ID, true
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
	if err := fn((&add).Receivers()...); err != nil {
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

func structDb(t *testing.T) RDB {
	var out, w io.Writer
	if trace {
		w = os.Stdout
	}
	if testing.Verbose() {
		fmt.Println("we are verbose")
		out = os.Stdout
	}
	dbs, err := NewRqlite("http://rbox1:4001", out, w)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dbs.dbs.Write(canned()); err != nil {
		t.Fatal(err)
	}
	if testing.Verbose() {
		dbs.debug = true
	}
	return dbs
}

func TestMain(m *testing.M) {
	flag.BoolVar(&trace, "trace", false, "trace rqlite calls")
	flag.Parse()
	os.Exit(m.Run())
}

func TestLoadBy(t *testing.T) {
	db := structDb(t)
	s := &testStruct{
		Name: "Waldo",
		Kind: 1982,
		Data: testData,
	}
	if err := db.Add(s); err != nil {
		t.Fatal(err)
	}
	if s.ID == 0 {
		t.Fatal("primary ID not created")
	}
	f := &testStruct{
		ID: s.ID,
	}
	// verify we can load what was saved
	if err := db.LoadBy(f, s.KeyFields()[0], f.ID); err != nil {
		t.Error(err)
	}
	// verify we don't load what doesn't exist
	if err := db.LoadBy(f, s.KeyFields()[0], -12345); err == nil {
		t.Fatal("expected error but there was none")
	} else {
		t.Logf("got expected error: %v\n", err)
	}
	t.Log("BY ID", f)
	u := testStruct{}
	if err := db.LoadBy(&u, "name", s.Name); err != nil {
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
	if err := db.LoadSelf(&s); err != nil {
		t.Error(err)
	}
	t.Log("BY SELF", s)
}

var testData = "lorem ipsum"

func TestDBObject(t *testing.T) {
	db := structDb(t)
	s := &testStruct{
		Name: "Grammatic, Bro",
		Kind: 2001,
		Data: testData,
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
	if err := db.Load(&z, m); err != nil {
		t.Fatal(err)
	}

	if err := db.Delete(s); err != nil {
		t.Fatal(err)
	}
}

func testRDB(t *testing.T) *sql.DB {
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

func TestAdd(t *testing.T) {
	const unique = "yoowho"
	db := structDb(t)
	s1 := &testStruct{
		Name:     unique,
		Kind:     2,
		Modified: time.Now().Truncate(time.Second).UTC(),
	}
	if err := db.Add(s1); err != nil {
		t.Fatal(err)
	}
	s2 := &testStruct{
		ID: s1.ID,
	}
	if err := db.LoadSelf(s2); err != nil {
		t.Fatal(err)
	}
	if err := s1.equal(s2); err != nil {
		t.Fatalf("\nerror: %v\n\nexpected: %+v\n but got: %+v\n", err, *s1, *s2)
	}
}

func TestUpdate(t *testing.T) {
	const unique = "heynow"
	db := structDb(t)
	orig := &testStruct{
		Name:     unique,
		Kind:     2,
		Modified: time.Now().Truncate(time.Second).UTC(),
	}
	if err := db.Add(orig); err != nil {
		t.Fatal(err)
	}
	orig.Kind = 2016
	orig.Name = "updated"
	if err := db.Update(orig); err != nil {
		t.Fatal(err)
	}

	dupe := &testStruct{
		ID: orig.ID,
	}
	if err := db.LoadSelf(dupe); err != nil {
		t.Fatal(err)
	}
	if err := orig.equal(dupe); err != nil {
		t.Fatalf("\nerror: %v\n\nexpected: %+v\n but got: %+v\n", err, *orig, *dupe)
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
