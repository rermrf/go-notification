package sqlx

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JsonColumn[T any] struct {
	Val   T
	Valid bool
}

func (j *JsonColumn[T]) Scan(src any) error {
	var bs []byte
	switch val := src.(type) {
	case nil:
		return nil
	case []byte:
		bs = val
	case string:
		bs = []byte(val)
	default:
		return fmt.Errorf("不支持 src 类型 %v", src)
	}
	if err := json.Unmarshal(bs, &j.Val); err != nil {
		return err
	}
	j.Valid = true
	return nil
}

func (j *JsonColumn[T]) Value() (driver.Value, error) {
	if !j.Valid {
		return nil, nil
	}
	res, err := json.Marshal(j.Val)
	return res, err
}
