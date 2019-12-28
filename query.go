package mte

import (
	gm "github.com/go-ginger/models"
)

type builder struct {
	Queries []interface{}
}

func (q *builder) Iterate(filters *gm.Filters, temp ...string) {

}
