package template

import "go-notification/internal/pkg/ginx"

const (
	SYSTEMERRORCODE = 506001
)

var (
	SystemError = ErrorCode{
		Code: SYSTEMERRORCODE,
		Msg:  "系统错误",
	}

	systemErrorResult = ginx.Result{
		Code: SystemError.Code,
		Msg:  SystemError.Msg,
	}
)

type ErrorCode struct {
	Code int
	Msg  string
}
