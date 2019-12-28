package mte

import (
	"bytes"
	"encoding/gob"
)

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
}

func DeepCopy(m map[string]interface{}) (result map[string]interface{}, err error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err = enc.Encode(m)
	if err != nil {
		return nil, err
	}
	err = dec.Decode(&result)
	if err != nil {
		return nil, err
	}
	return
}
