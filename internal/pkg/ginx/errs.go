package ginx

import (
	"errors"
)

var ErrUnauthorized = errors.New("unauthorized")
var ErrSessionKeyNotFound = errors.New("session key not found")

var ErrNoResponse = errors.New("no response")
