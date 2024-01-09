package database

import "context"

type RowsAPI interface {
	Close() error
	ColumnTypes(...string) ([]*ColumnType, error)
	Columns(...string) ([]string, error)
	Data(...string) []interface{}
	DisplayData(...string) []interface{}
	Field(string) interface{}
	FieldByIndex(int) interface{}
	FieldIndex(string) int
	Err() error
	Next() bool
	Context() context.Context
	//NextResultSet() bool
	Scan(castTypeVal func(valToCast interface{}, colType interface{}) (val interface{}, scanned bool)) error
}
