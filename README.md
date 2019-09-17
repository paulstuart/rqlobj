# rqlobj [![GoDoc](https://godoc.org/github.com/paulstuart/rqlobj?status.svg)](http://godoc.org/github.com/paulstuart/rqldbj)
An Object Relational Manager for rqlite

Because rqlite does not provide the standard `*sql.DB connection`, it will not work with existing ORMs, which all wrap around that interface.

This project allows for using struct tags to annotate your structs that need persistence. The associated `dbgen` command is used to
generate data handlers for these struts to enable CRUD and list operations against objects without writing any SQL.

