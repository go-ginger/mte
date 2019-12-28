package mte

type Root struct {
	Items []interface{}
}

func (r *Root) Add(item interface{}) {
	r.Items = append(r.Items, item)
}
