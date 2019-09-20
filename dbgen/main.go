// Copyright 2014 The Go Authors. All rights reserved.
// Copyright 2015 Paul Stuart. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from golang.org/x/tools/cmd/stringer/stringer.go

// dbgen is a tool to automate the creation of CRUD methods
// (create/update/delete) that satisfy the DBObject interface
// in github.com/paulstuart/rqlobj.
//
// For example, given this snippet,
//
//	package dbobjs
//
// type User struct {
// 	ID       int64		`sql:"id,key" table:"users"`
// 	Username string		`sql:"username"`
// 	First    string		`sql:"firstname"`
// 	Last     string		`sql:"lastname"`
// 	Email    string		`sql:"email"`
// 	Role     int		`sql:"role"`
// 	UserID   int64		`sql:"userid"    audit:"user"`
// 	Modified time.Time  `sql:"modified"  audit:"time"`
// 	Created  time.Time  `sql:"created"  update="false"
// }
//
// running this command
//
//	dbgen
//
// in the same directory will create the file db_generated.go, in package dbobjs,
// containing the definition:
//
//
// Typically this process would be run using go generate, like this:
//
//	//go:generate dbgen
//
// The -type flag accepts a comma-separated list of types so a single run can
// generate methods for multiple types. The default output file is db_generated.go,
// where t is the lower-cased name of the first type listed. It can be overridden
// with the -output flag.
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// For testing
//go:generate ./dbgen -output generated_test.go -type testStruct struct_test.go
var (
	tagName   = flag.String("tag", tagDefault, "the tag used to annotate structs")
	typeNames = flag.String("type", "", "comma-separated list of type names; leave blank for all")
	output    = flag.String("output", "", "output file name; default is "+generatedFile)
	prefix    = flag.String("prefix", "", "only convert types with the given prefix")
	verbose   = flag.Bool("verbose", false, "show processing info")
)

const (
	generatedFile = "db_generated.go"
	ignore        = "github.com/paulstuart/rqlobj.DBObject"
	tagDefault    = "sql"
)

// Usage is a replacement usage function for the flags package.
func Usage() {
	const msg = `
Usage of %s:

%s [flags] [-type T] files... # Must be a single package

For more information, see: http://github.com/paulstuart/rqlobj/dbgen

Flags:
`

	fmt.Fprintf(os.Stderr, msg, os.Args[0], os.Args[0])
	/*
		// TODO: rethink applying options
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\tdbgen [flags] [-type T] [directory]\n")
		fmt.Fprintf(os.Stderr, "\tdbgen [flags[ [-type T] files... # Must be a single package\n")
		fmt.Fprintf(os.Stderr, "For more information, see:\n")
		fmt.Fprintf(os.Stderr, "\thttps://github.com/paulstuart/dbobj/tree/master/dbgen\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	*/
	os.Exit(2)
}

type SQLInfo struct {
	Name      string              // type name
	Table     string              // sql table
	KeyNames  []string            // member name for key
	KeyFields []string            // sql field for key
	UserField string              // sql field for user id
	TimeField string              // sql field for timestamp
	Order     []string            // sql fields in order
	Types     []string            // data types in order
	Fields    map[string]string   // map of struct tag to column name
	NoUpdate  map[string]struct{} // set of fields that should not be updated
	Primary   bool                // there is one key and it is an int64
	FK        map[string]string   // foreign key: field -> table(field)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("dbgen: ")
	//flag.Usage = Usage
	flag.Parse()
	names := strings.Split(*typeNames, ",")

	// We accept either one directory or a list of files. Which do we have?
	args := flag.Args()
	if len(args) == 0 {
		// Default: process whole package in current directory.
		args = []string{"."}
	}

	// Parse the package once.
	var (
		dir string
		g   Generator
	)
	if len(args) == 1 && isDirectory(args[0]) {
		g.parsePackageDir(args[0])
	} else {
		dir = filepath.Dir(args[0])
		g.parsePackageFiles(args)
	}

	// Print the header and package clause.
	var cmdargs string
	if len(os.Args) > 1 {
		cmdargs = " " + strings.Join(os.Args[1:], " ")
	}
	g.Printf("// generated by '%s%s'; DO NOT EDIT\n", path.Base(os.Args[0]), cmdargs)
	g.Printf("\npackage %s\n", g.pkg.name)

	if len(names) == 0 {
		g.generate("")
	} else {
		for _, typeName := range names {
			g.generate(typeName)
		}
	}

	// go fmt the output.
	src := g.format()

	// Write to file.
	outputName := *output
	if outputName == "" {
		outputName = filepath.Join(dir, generatedFile)
	}
	err := ioutil.WriteFile(outputName, src, 0644)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}
}

// isDirectory reports whether the named file is a directory.
func isDirectory(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		log.Println(err)
		return false
	}
	return info.IsDir()
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
// sql tag added for testing
type Generator struct {
	buf bytes.Buffer `sql:"buf" table:"generator"` // Accumulated output.
	pkg *Package     // Package we are scanning.
}

func (g *Generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

// File holds a single parsed file and associated data.
type File struct {
	pkg  *Package  // Package to which this file belongs.
	file *ast.File // Parsed AST.
	// These fields are reset for each type being generated.
	TypeName string     // Name of the current type.
	findName string     // Type name to match (if set)
	values   []*SQLInfo // Accumulator for sql annotated objects
}

// Package has sql tags for testing
type Package struct {
	dir      string    `sql:"pkgdir,key" table:"pkg"`
	name     string    `sql:"name" audit:"name"`
	fakeTime time.Time `sql:"fake"`
	defs     map[*ast.Ident]types.Object
	files    []*File
	typesPkg *types.Package
}

// parsePackageDir parses the package residing in the directory.
func (g *Generator) parsePackageDir(directory string) {
	pkg, err := build.Default.ImportDir(directory, 0)
	if err != nil {
		log.Fatalf("cannot process directory %s: %s", directory, err)
	}
	var names []string
	names = append(names, pkg.GoFiles...)
	names = append(names, pkg.CgoFiles...)
	names = append(names, pkg.SFiles...)
	names = prefixDirectory(directory, names)
	g.parsePackage(directory, names, nil)
}

// parsePackageFiles parses the package occupying the named files.
func (g *Generator) parsePackageFiles(names []string) {
	//fmt.Println("PARSE", names)
	g.parsePackage(".", names, nil)
}

// prefixDirectory places the directory name on the beginning of each name in the list.
func prefixDirectory(directory string, names []string) []string {
	if directory == "." {
		return names
	}
	ret := make([]string, len(names))
	for i, name := range names {
		ret[i] = filepath.Join(directory, name)
	}
	return ret
}

// parsePackage analyzes the single package constructed from the named files.
// If text is non-nil, it is a string to be used instead of the content of the file,
// to be used for testing. parsePackage exits if there is an error.
//
// returns true if "time" package is required (TODO: rethink that time stuff)
func (g *Generator) parsePackage(directory string, names []string, text interface{}) bool {
	var files []*File
	var astFiles []*ast.File
	g.pkg = new(Package)
	fs := token.NewFileSet()
	for _, name := range names {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		status("evaluating file: %s\n", name)
		parsedFile, err := parser.ParseFile(fs, name, text, 0)
		if err != nil && name != generatedFile {
			log.Fatalf("parsing package: %s: %s", name, err)
		}
		astFiles = append(astFiles, parsedFile)
		files = append(files, &File{
			file: parsedFile,
			pkg:  g.pkg,
		})
	}
	if len(astFiles) == 0 {
		log.Fatalf("%s: no buildable Go files", directory)
	}
	g.pkg.name = astFiles[0].Name.Name
	g.pkg.files = files
	g.pkg.dir = directory
	// Type check the package.
	// Skipping for now as looking to be unneeded (TODO: affirm and delete if so)
	//g.pkg.check(fs, astFiles)
	return false
}

// TODO: rethink this. For now, assume it's all good
// check type-checks the package. The package must be OK to proceed.
func (pkg *Package) check(fs *token.FileSet, astFiles []*ast.File) {
	pkg.defs = make(map[*ast.Ident]types.Object)
	config := types.Config{
		Importer: importer.Default(),
		//Importer: importer.For("gc", nil),
		//Importer: importer.For("source", nil),
		Error: func(e error) {
			if err, ok := e.(types.Error); ok {
				fmt.Printf("PKG ERR (%T): %+v\n", e, e)
				err.Msg = ""
				e = nil
				return
			}
			err := e.(types.Error)
			i := strings.Index(err.Msg, "DBObject")
			if strings.HasSuffix(err.Msg, ignore) || i > 0 {
				return
			}
			// TODO: switch on err type rather than error content
			if strings.Index(err.Msg, "has no field or method") > 0 {
				switch {
				case strings.Index(err.Msg, "TableName") > 0:
				default:
					file := err.Fset.File(err.Pos)
					log.Println("POS:", err.Pos, "MSG:", err.Msg, "INDEX:", i, "SOFT:", err.Soft, "FILE:", file.Name())
					return
				}
				err.Msg = ""
				e = nil
			}
		},
	}
	info := &types.Info{
		Defs: pkg.defs,
	}
	typesPkg, err := config.Check(pkg.dir, fs, astFiles, info)
	if err != nil {
		log.Println("failed checking package:", err)
	}
	pkg.typesPkg = typesPkg
}

// generate produces the DBObject methods for the named type.
func (g *Generator) generate(typeName string) bool {
	for _, file := range g.pkg.files {
		file.findName = typeName
		if file.file != nil {
			ast.Inspect(file.file, file.genDecl)
			for _, v := range file.values {
				//fmt.Printf("GEN (%T): %+v\n", v, v)
				g.buildWrappers(v)
			}
		}
	}
	return false
}

// format returns the gofmt-ed contents of the Generator's buffer.
func (g *Generator) format() []byte {
	src, err := format.Source(g.buf.Bytes())
	if err != nil {
		// Should never happen, but can arise when developing this code.
		// The user can compile the output to see the error.
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		return g.buf.Bytes()
	}
	return src
}

func status(msg string, args ...interface{}) {
	if *verbose {
		log.Printf(msg, args...)
	}
}

//
// Parse the tags, build tables of the metadata
//
func sqlTags(typeName string, fields *ast.FieldList) *SQLInfo {
	if *prefix != "" && !strings.HasPrefix(typeName, *prefix) {
		const msg = "skipping type %q as it does not have prefix: %q\n"
		status(msg, typeName, *prefix)
		return nil
	}
	if typeName == "" {
		status("skipping blank type name")
		return nil
	}
	status("evaluating type %q for sql tags\n", typeName)
	info := SQLInfo{
		Fields:   make(map[string]string), // [memberName]sqlName
		Order:    make([]string, 0, len(fields.List)),
		NoUpdate: make(map[string]struct{}),
		FK:       make(map[string]string),
	}
	good := false
	for _, field := range fields.List {
		if t := field.Tag; t != nil {
			name := string(field.Names[0].Name)
			s := string(t.Value)
			// the code uses backticks to metaquote, need to strip them whilst evaluating
			tag := reflect.StructTag(s[1 : len(s)-1])
			if sql := tag.Get(*tagName); sql != "" {
				typ := fmt.Sprint(field.Type)
				//fmt.Printf("FLD NAME: %q TYPE: %q\n", field.Names[0].Name, typ)
				switch typ {
				case "string":
					info.Types = append(info.Types, "text")
				case "&{time Time}":
					info.Types = append(info.Types, "datetime")
				case "int", "int64", "uint", "uint64":
					info.Types = append(info.Types, "integer")
				default:
					info.Types = append(info.Types, "text")
				}
				if table := tag.Get("table"); len(table) > 0 {
					info.Table = table
				}

				// identify its name and if key field
				var hasKey bool
				parts := strings.Split(sql, ",")
				if len(parts) > 1 {
					sql = parts[0]
					// using default "sql" for struct tags
					if *tagName == tagDefault {
						if parts[1] == "key" {
							hasKey = true
							const msg = "type: %s field: %s has is a key\n"
							status(msg, name, sql)
						} else {
							log.Println("invalid option following field name:", parts[1])
							// TODO: any more effort to transmit the error? Panic?
						}
					}
				}
				if !hasKey {
					// if not using "sql" as the struct tag,
					// a fallback option of using key=true,
					// just as "table" gets a seperate mention
					if key := tag.Get("key"); key != "" {
						hasKey, _ = strconv.ParseBool(key)
					}
				}
				if hasKey {
					if typ == "int64" {
						if !info.Primary {
							info.Primary = true
						} else {
							// more than one complicates things
							info.Primary = false
							const msg = "type: %s field: %s -- breaks prior primary key\n"
							status(msg, name, sql)
						}
					} else {
						info.Primary = false
					}
					info.KeyNames = append(info.KeyNames, name)
					info.KeyFields = append(info.KeyFields, sql)
					// TODO: is NoUpdate simply the intersection of keys & selects?
					info.NoUpdate[sql] = struct{}{}
				} else {
					info.Fields[name] = sql
					info.Order = append(info.Order, name)
					//info.Types = append(info.Types, field.Type)
				}
				// look for foreign key declarations
				if fk := tag.Get("fk"); fk != "" {
					const msg = "type: %s field: %s has foreign key: %s\n"
					status(msg, name, sql, fk)
					info.FK[sql] = fk
				}
				good = true
			}
			// note fields to be excluded from object update queries
			if update := tag.Get("update"); len(update) > 0 {
				if up, err := strconv.ParseBool(update); err == nil && up == false {
					info.NoUpdate[field.Names[0].Name] = struct{}{}
				}
			}
		}
	}
	if good {
		//fmt.Printf("INFO: %+v\n", info)
		return &info
	}
	return nil
}

// genDecl processes a declaration clause.
func (f *File) genDecl(node ast.Node) bool {
	switch x := node.(type) {
	case *ast.TypeSpec:
		f.TypeName = x.Name.Name
	case *ast.StructType:
		if len(f.findName) == 0 || f.findName == f.TypeName {
			if tags := sqlTags(f.TypeName, x.Fields); tags != nil {
				tags.Name = f.TypeName
				f.values = append(f.values, tags)
			}
			return false
		}
	}
	return true
}

// buildWrappers generates the variables and String method for a single run of contiguous values.
func (g *Generator) buildWrappers(s *SQLInfo) {
	var insert_fields, names, elem, ptr, set, sql []string
	keyField := make(map[string]struct{})
	// fields for sql keys and regular are presented seperately. join them.
	sql = append(sql, s.KeyFields...)
	for i, name := range s.KeyNames {
		ptr = append(ptr, "&o."+name)
		keyField[s.KeyFields[i]] = struct{}{}
	}
	for _, k := range s.Order {
		if k != "" {
			v := s.Fields[k]
			sql = append(sql, v)
			names = append(names, `"`+k+`"`)
			elem = append(elem, "o."+k)
			ptr = append(ptr, "&o."+k)
			set = append(set, v+"=?")
			if _, ok := s.NoUpdate[v]; !ok {
				insert_fields = append(insert_fields, v)
			}
		}
	}
	g.Printf("\n\n//\n// %s DBObject generator\n//\n", s.Name)
	g.Printf(metaNewObj, s.Name)
	g.Printf("\n//\n// %s DBObject interface functions\n//\n", s.Name)
	if s.Primary {
		g.Printf(metaPrimaryValid, s.Name, s.KeyNames[0])
	} else {
		g.Printf(metaPrimaryInvalid, s.Name)
	}
	g.Printf(metaInsertValues, s.Name, strings.Join(elem, ","))
	for _, name := range s.KeyNames {
		elem = append(elem, "o."+name)
	}
	g.Printf(metaUpdateValues, s.Name, strings.Join(elem, ","))
	g.Printf(metaReceivers, s.Name, strings.Join(ptr, ","))
	kv := make([]string, len(s.KeyNames))
	for i, name := range s.KeyNames {
		kv[i] = "o." + name
	}
	if len(s.KeyNames) > 0 {
		g.Printf(metaKeyValues, s.Name, strings.Join(kv, ","))
	} else {
		g.Printf(metaKeyless, s.Name)
	}
	if s.Primary {
		g.Printf(metaSetPrimary, s.Name, s.KeyNames[0])
	} else {
		g.Printf(metaSetNo, s.Name)
	}

	status("%s uses table %s\n", s.Name, s.Table)
	g.Printf(metaSQLGet, s.Name, s.Table, strings.Join(sql, ","))
	g.Printf(metaSQLResults, s.Name, s.Table)
	g.Printf(metaTableName, s.Name, s.Table)
	g.Printf(metaSelectFields, s.Name, strings.Join(sql, ","))
	g.Printf(metaInsertFields, s.Name, strings.Join(insert_fields, ","))
	fields := quoteList(s.KeyFields)
	g.Printf(metaKeyFields, s.Name, fields)
	keyNames := quoteList(s.KeyNames)
	g.Printf(metaKeyNames, s.Name, keyNames)
	g.Printf(metaElements, s.Name, qList(names))

	// TODO: add support for default values <======================================================= SOON!

	g.Printf(metaSQLCreate, s.Name, s.Table, rowString(sql, s.Types, keyField, s.FK, s.Primary), "`")
}

// TODO: apply this struct for enhanced table generation
type Field struct {
	Check   string
	Default string // make interface{} ?
	Unique  bool
}

// convert a list of column defs to a string
// TODO: generate indexes for tables with multiple keys
func rowString(fields, types []string, keys map[string]struct{}, fk map[string]string, primary bool) string {
	var buf strings.Builder
	if len(fields) != len(types) {
		const msg = "slice sizes don't match for fields:%d -- types:%d\n"
		return fmt.Sprintf(msg, len(fields), len(types))
	}
	for i, field := range fields {
		if i > 0 {
			buf.WriteString(",\n")
		}
		buf.WriteString("  ")
		buf.WriteString(field)
		buf.WriteString(" ")
		buf.WriteString(types[i])
		if _, ok := keys[field]; ok && len(keys) == 1 {
			if (primary || len(keys) > 1) && i == 0 {
				buf.WriteString(" primary key")
			}
		}
		// foreign key support:
		// fieldName fieldType  REFERENCES artist(artistid) ON UPDATE CASCADE;
		if ref, ok := fk[field]; ok {
			status("field: %s applying fk: %s\n", field, ref)
			buf.WriteString(" REFERENCES ")
			buf.WriteString(ref)
			buf.WriteString(" ON UPDATE CASCADE")
		}

	}
	return buf.String()
}

func qList(list []string) string {
	return strings.Join(list, ",")
}

func quoteList(list []string) string {
	var b strings.Builder
	for i, item := range list {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(item)
		b.WriteByte('"')
	}
	return b.String()
}

// Arguments to format are:
//	[1]: type name
//	[2]: sql table
//	[3]: comma separated list of fields
//	[4]: comma separated list of parameter placeholders, e.g., (?,?,?)
const metaReplace = `func (o *%[1]s) ReplaceQuery() string {
	return "replace into %[2]s (%[3]s) values(%[4]s)"
}

`

/*
// Arguments to format are:
//	[1]: type name
//	[2]: sql table
//	[3]: comma separated list of fields
//	[4]: comma separated list of parameter placeholders, e.g., (?,?,?)
const metaInsertQuery = `func (o *%[1]s) InsertQuery() string {
	return "insert into %[2]s (%[3]s) values(%[4]s)"
}

`
*/

// Arguments to format are:
//	[1]: type name
//	[2]: sql table
//	[3]: update set pairs
//	[4]: where criteria
const metaUpdate = `func (o *%[1]s) UpdateQuery() string {
	return "update %[2]s set %[3]s where %[4]s"
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: sql table
//	[3]: insert fields (excluding key)
const metaInsertValues = `func (o *%[1]s) InsertValues() []interface{} {
	return []interface{}{%s}
}

`

// metaUpdateValues arguments
//	[1]: type name
//	[2]: sql table
//	[3]: update fields (including key)
const metaUpdateValues = `func (o *%[1]s) UpdateValues() []interface{} {
	return []interface{}{%s}
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: sql table
//	[3]: update fields (including key)
const metaReceivers = `func (o *%[1]s) Receivers() []interface{} {
	return []interface{}{%s}
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: key fields, e.g. o.ID,o.Name,o.Kind
const metaKeyValues = `func (o *%[1]s) KeyValues() []interface{} {
	return []interface{}{%[2]s}
}

`

// Arguments to format are:
//	[1]: type name
const metaKeyless = `func (o *%[1]s) Keys() []interface{} {
	return nil
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: key field
const metaSetPrimary = `func (o *%[1]s) SetPrimary(id int64) {
	o.%[2]s = id
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: key field
const metaSetNo = `func (o *%[1]s) SetPrimary(_ int64) {
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: table name
const metaTableName = `func (o *%[1]s) TableName() string {
	return "%[2]s"
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: key field
const metaKeyFields = `func (o *%[1]s) KeyFields() []string {
	return []string{%[2]s}
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: key name
const metaKeyNames = `func (o *%[1]s) KeyNames() []string {
	return []string{%[2]s}
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: sql table
//	[3]: where criteria
const metaDelete = `func (o *%[1]s) DeleteQuery() string {
	return "delete from %[2]s where %[3]s"
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: select fields
const metaSelectFields = `func (o *%[1]s) SelectFields() string {
	return "%[2]s"
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: insert fields
const metaInsertFields = `func (o *%[1]s) InsertFields() string {
	return "%[2]s"
}

`

// Arguments to format are:
//	[1]: type name
const metaNewObj = `func (o %[1]s) NewObj() interface{} {
	return new(%[1]s)
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: member names
const metaElements = `func (o *%[1]s) Elements() []string {
	return []string{%[2]s}
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: key member name
const metaPrimaryValid = `func (o *%[1]s) Primary() (int64, bool) {
	return o.%[2]s, true
}

`

// Arguments to format are:
//	[1]: type name
const metaPrimaryInvalid = `func (o *%[1]s) Primary() (int64, bool) {
	return 0, false
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: table name
//	[3]: select fields
//			 1	 2	  3			  4
const metaSQLGet = `

type %[1]ss []%[1]s

func (o *%[1]ss) SQLGet(extra string) string {
	return "select %[3]s from %[2]s " + extra + ";"
}

`

/*
	// funky old way to append a row
	*o = append(*o, %[1]s{})
	off := len(*o) - 1
	dest := &((*o)[off])
	ptrs := dest.Receivers()
	return fn(ptrs...)
*/

// Arguments to format are:
//	[1]: type name
//	[2]: table name
//			 1	 2	  3			  4
const metaSQLResults = `

// SQLResults takes the equivalent of the Scan function in database/sql
func (o *%[1]ss) SQLResults(fn func(...interface{}) error) error {
	var add %[1]s
	if err := fn((&add).Receivers()...); err != nil {
	    return err
	}
	*o = append(*o, add)
	return nil
}

`

// Arguments to format are:
//	[1]: type name
//	[2]: table name
//	[3]: column declarations
//	[4]: "`" to cheat at nesting quotes
//			 1	 2	  3
const metaSQLCreate = `


// SQLCreate returns a query to create a table for the object
func (o *%[1]s) SQLCreate() string {
	return %[4]screate table if not exists %[2]s (
%[3]s
);%[4]s
}

`

//
// these structs are for testing `dbgen` against
//
type hasPrimary struct {
	ID      int64     `sql:"id,key" table:"teststruct"`
	Name    string    `sql:"name"`
	Kind    int       `sql:"kind"`
	Data    []byte    `sql:"data"`
	Created time.Time `sql:"created" update:"false"`
}

type hasMany struct {
	ID      int64     `sql:"id,key" table:"teststruct"`
	Family  string    `sql:"family,key"`
	Name    string    `sql:"name"`
	Kind    int       `sql:"kind"`
	Data    []byte    `sql:"data"`
	Created time.Time `sql:"created" update:"false"`
}

type hasMulti struct {
	ID      int64     `sql:"id,key" table:"teststruct"`
	Sec     int64     `sql:"other_key,key"`
	Name    string    `sql:"name"`
	Kind    int       `sql:"kind"`
	Data    []byte    `sql:"data"`
	Created time.Time `sql:"created" update:"false"`
}
