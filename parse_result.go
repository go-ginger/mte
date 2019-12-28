package mte

import (
	dl "github.com/go-ginger/dl/query"
)

type ParseResult struct {
	dl.IParseResult

	query interface{}
	sort  interface{}
}

func (r *ParseResult) GetQuery() (query interface{}) {
	return r.query
}

func (r *ParseResult) GetSort() (sort interface{}) {
	return r.sort
}
