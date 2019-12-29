package mte

import (
	dl "github.com/go-ginger/dl/query"
	gm "github.com/go-ginger/models"
)

type Parser struct {
	dl.IParser

	QueryTemplates map[string]map[string]interface{}
	QueryFillPaths map[string][][]string
}

var operators map[string]bool
var conditions map[string]bool
var rangeConditions map[string]bool

func init() {
	operators = map[string]bool{"$and": true, "$or": true, "$all": true, "$in": true, "$ne": true}
	rangeConditions = map[string]bool{"$lt": true, "$lte": true, "$gt": true, "$gte": true}
	conditions = map[string]bool{"$lt": true, "$lte": true, "$gt": true, "$gte": true, "$ne": true}
}

func (p *Parser) setNested(data interface{}, value interface{}, paths ...string) {
	if data != nil && paths != nil {
		if paths[0] == "$" {
			items := data.([]interface{})
			for _, item := range items {
				p.setNested(item.(map[string]interface{}), value, paths[1:]...)
			}
			return
		}
		if paths[0] == "*" {
			items := data.([]interface{})
			dataMap := data.(map[string]interface{})
			for _, key := range items {
				p.setNested(dataMap[key.(string)].(map[string]interface{}), value, paths[1:]...)
			}
			return
		}
		path := paths[0]
		if path != "" {
			dataMap := data.(map[string]interface{})
			if len(paths) == 1 {
				firstData := dataMap[path]
				if f, ok := firstData.(func(interface{}) interface{}); ok {
					value = f(value)
				}
				dataMap[path] = value
				return
			}
			p.setNested(dataMap[path], value, paths[1:]...)
		}
	}
	return
}

func (p *Parser) addTemplate(fieldName string, array []interface{}, value interface{}) (query interface{}) {
	if qTemplate, ok := p.QueryTemplates[fieldName]; ok {
		if qTemplate == nil {
			return
		}
		qTemplate, err := DeepCopy(qTemplate)
		if err != nil {
			return
		}
		queryFillPaths := p.QueryFillPaths[fieldName]
		for _, queryFillPath := range queryFillPaths {
			p.setNested(qTemplate, value, queryFillPath...)
		}
		array = append(array, qTemplate)
	} else {
		array = append(array, map[string]interface{}{
			"match": map[string]interface{}{
				fieldName: value,
			},
		})
	}
	query = array
	return
}

func (p *Parser) generateOperator(op string, field ...interface{}) (query map[string]interface{},
	value *[]interface{}) {
	value = &[]interface{}{}
	switch op {
	case "$and", "$all":
		query = map[string]interface{}{
			"bool": map[string]interface{}{
				"must": value,
			},
		}
		break
	case "$or", "$in":
		query = map[string]interface{}{
			"bool": map[string]interface{}{
				"should": value,
			},
		}
		break
	case "$ne":
		query = map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{
						"bool": map[string]interface{}{
							"must_not": value,
						},
					},
					map[string]interface{}{
						"bool": map[string]interface{}{
							"must_not": []interface{}{
								map[string]interface{}{
									"exists": map[string]interface{}{
										"field": field[0],
									},
								},
							},
						},
					},
				},
			},
		}
		break
	}
	return
}

func (p *Parser) generateCondition(data map[string]interface{}, op, key string, value interface{}) {
	if len(data) == 0 {
		if _, isRange := rangeConditions[op]; isRange {
			data["range"] = map[string]interface{}{
				key: map[string]interface{}{},
			}
		}
	}
	switch op {
	case "$lt":
		data["range"].(map[string]interface{})[key].(map[string]interface{})["lt"] = value
		break
	case "$lte":
		data["range"].(map[string]interface{})[key].(map[string]interface{})["lte"] = value
		break
	case "$gt":
		data["range"].(map[string]interface{})[key].(map[string]interface{})["gt"] = value
		break
	case "$gte":
		data["range"].(map[string]interface{})[key].(map[string]interface{})["gte"] = value
		break
	case "$exists":
		data["exists"] = map[string]interface{}{
			"field": key,
		}
		break
	}
}

func (p *Parser) iterate(data interface{}, temp ...string) (queries []interface{}) {
	var tempValue string
	if temp != nil {
		tempValue = temp[0]
	}
	queries = make([]interface{}, 0)
	if q, ok := data.(gm.Filters); ok {
		opMap := map[string]interface{}{}
		for k, v := range q {
			if qTemplate, ok := p.QueryTemplates[k]; ok && qTemplate == nil {
				continue
			}
			if v == nil {
				continue
			}
			if _, isOp := operators[k]; isOp {
				qo, a := p.generateOperator(k, tempValue)
				if list, isList := v.([]gm.Filters); isList {
					for _, listItem := range list {
						for _, item := range p.iterate(listItem, tempValue) {
							*a = append(*a, item)
						}
					}
				} else if list, isList := v.([]interface{}); isList {
					for _, listItem := range list {
						for _, item := range p.iterate(listItem, tempValue) {
							*a = append(*a, item)
						}
					}
				} else {
					for _, item := range p.iterate(v, tempValue) {
						*a = append(*a, item)
					}
				}
				queries = append(queries, qo)
			} else if _, isCond := conditions[k]; isCond {
				exists := true
				if _, ok := opMap[tempValue]; !ok {
					exists = false
					opMap[tempValue] = &map[string]interface{}{}
				}
				p.generateCondition(opMap[tempValue].(map[string]interface{}), k, tempValue, v)
				if !exists {
					queries = append(queries, opMap[tempValue])
				}
			} else if k == "$exists" {
				p.generateCondition(opMap, k, tempValue, v)
				queries = append(queries, opMap)
			} else if _, isMap := v.(*map[string]interface{}); isMap {
				queries = append(queries, p.iterate(v, k)...)
			} else {
				templateQuery := p.addTemplate(k, queries, v)
				if templateQuery != nil {
					queries = append(queries, templateQuery)
				}
			}
		}
	} else {
		if tempValue != "" {
			templateQuery := p.addTemplate(tempValue, queries, data)
			if templateQuery != nil {
				queries = append(queries, templateQuery)
			}
		}
	}
	return
}

func (p *Parser) Parse(request gm.IRequest) (result dl.IParseResult) {
	req := request.GetBaseRequest()
	pResult := &ParseResult{}
	// sort
	if req.Sort != nil {
		sortResult := make([]string, 0)
		for _, sort := range *req.Sort {
			s := sort.Name
			if sort.Ascending {
				s += ":asc"
			} else {
				s += ":desc"
			}
			sortResult = append(sortResult, s)
		}
		sortResult = append(sortResult, "_score:desc")
		pResult.sort = sortResult
	}
	// query
	must := make([]interface{}, 0)
	queries := p.iterate(*req.Filters)
	for _, q := range queries {
		must = append(must, q)
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"bool": map[string]interface{}{
							"must": must,
						},
					},
				},
			},
		},
	}
	pResult.query = query
	result = pResult
	return
}
