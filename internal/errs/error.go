package errs

import "errors"

var (
	// 业务错误
	ErrInvalidParameter = errors.New("参数错误")
)
