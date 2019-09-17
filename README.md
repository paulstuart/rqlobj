# rqlobj [![GoDoc](https://godoc.org/github.com/paulstuart/rqlobj?status.svg)](http://godoc.org/github.com/paulstuart/rqldbj)[![Build Status](https://travis-ci.com/paulstuart/rqlobj.svg?branch=master)](https://travis-ci.com/paulstuart/rqlobj)
An Object Relational Manager (ORM) for rqlite

Because rqlite does not provide the standard `*sql.DB` connection, it will not work with existing ORMs, which all wrap around that interface.

This project allows for using struct tags to annotate your structs that need persistence. The associated `dbgen` command is used to
generate data handlers for these struts to enable CRUD and list operations against objects without writing any SQL.

The goals of this project are to provide object mapping, and it does not include schema generation or data migrations.
