package rqlobj

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

var (
	num   int = 23
	trace bool
	hey   string
)

func init() {
	/*
	 */
	testing.Init()
	flag.IntVar(&num, "num", num, "test ints")
	flag.StringVar(&hey, "hey", hey, "test flags")
	flag.BoolVar(&trace, "trace", true, "trace rqlite calls")
	flag.Parse()

	fmt.Println("INIT TRACE:", trace)
	fmt.Println("INIT HEY:", hey)
	fmt.Println("INIT NUM:", num)
}

type testStruct struct {
	ID       int64     `sql:"id" key:"true" table:"structs"`
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
	return "structs"
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
	const table = "struct"
	return fmt.Sprintf("select %s from %s", fields, table)
}

func (s *_testStruct) SQLResults(func(...interface{}) error) error {
	return nil
}

type testStrings struct {
	ID       int64     `sql:"id" key:"true" table:"structs"`
	Name     string    `sql:"name"`
	Kind     int       `sql:"kind"`
	Data     string    `sql:"data"`
	Modified time.Time `sql:"modified" update:"false"`
	astring  string
	anint    int
}

const queryCreate = `create table if not exists structs (
    id integer not null primary key,
    name text,
    kind int,
    data blob,
    modified DATETIME DEFAULT CURRENT_TIMESTAMP
);`

type testMap map[int64]testStruct

func structDb(t *testing.T) DBU {
	var w io.Writer
	if trace {
		w = os.Stdout
	}
	dbs, err := NewRqlite("http://localhost:4001", w)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dbs.dbs.Write(canned()); err != nil {
		t.Fatal(err)
	}
	return dbs
}

func TestMain(m *testing.M) {
	/*
		hey := "hey man"
		flag.StringVar(&hey, "hey", hey, "test flags")
		flag.BoolVar(&trace, "trace", true, "trace rqlite calls")
		flag.Parse()
		fmt.Println("MAIN TRACE:", trace)
		fmt.Println("MAIN HEY:", hey)
	*/
	os.Exit(m.Run())
}

func TestFindBy(t *testing.T) {
	db := structDb(t)
	s := testStruct{}
	if err := db.FindBy(&s, "name", "Bobby Tables"); err != nil {
		t.Error(err)
	}
	t.Log("BY NAME", s)
	u := testStruct{}
	if err := db.FindBy(&u, "id", 1); err != nil {
		t.Error(err)
	}
	t.Log("BY ID", u)
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
	if err := db.Save(s); err != nil {
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

func prepare(db *sql.DB) {
	const queryInsert = "insert into structs(name, kind, data) values(?,?,?)"
	db.Exec(queryCreate)
	db.Exec(queryInsert, "abc", 23, "what ev er")
	db.Exec(queryInsert, "def", 69, "m'kay")
	db.Exec(queryInsert, "ghi", 42, "meaning of life")
	db.Exec(queryInsert, "jkl", 2, "of a kind")
	db.Exec(queryInsert, "mno", 2, "of a drag")
	db.Exec(queryInsert, "pqr", 2, "of a sort")
}

func canned() []string {
	const queryInsert = "insert into structs(name, kind, data) values(%s)"
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

func errLogger(t *testing.T) chan error {
	e := make(chan error, 4096)
	go func() {
		for err := range e {
			t.Error(err)
		}
	}()
	return e
}

/*
type _testStruct []testStruct

func (list *_testStruct) add() {
	*list = append(*list, testStruct{})
}

func (list *_testStruct) Receivers() []interface{} {
	*list = append(*list, testStruct{})
	tl := *(*[]testStruct)(list)
	i := len(tl) - 1
	return tl[i].MemberPointers()
}

func (_ *_testStruct) QueryString(where string) string {
	var o testStruct
	if where == "" {
		return fmt.Sprintf("select %s from %s\n", o.SelectFields(), o.TableName())
	}
	return fmt.Sprintf("select %s from %s where %s\n", o.SelectFields(), o.TableName(), where)
}
*/

func TestListQuery(t *testing.T) {
	db := structDb(t)
	list := new(_testStruct)
	//db.ListQuery(list, "(id % 2) = 0")
	db.ListQuery(list, "")
	for _, item := range *list {
		t.Logf("ITEM:  %+v\n", item)
	}
}

func TestRqliteQuery(t *testing.T) {
	db := structDb(t)
	list := new(_testStruct)
	db.ListQuery(list, "(id % 2) = 0")
	for _, item := range *list {
		t.Logf("ITEM:  %+v\n", item)
	}
}
