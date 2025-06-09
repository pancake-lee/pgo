package perr

import "errors"

var (
	ErrParamInvalid = errors.New("param is invalid")
	ErrTokenSign    = errors.New("sign the token failed")
)
