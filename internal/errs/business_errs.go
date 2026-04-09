package errs

import "fmt"

type BusinessErr struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *BusinessErr) Error() string {
	return fmt.Sprintf("业务错误, code:%d, msg:%s", e.Code, e.Msg)
}
