package core

import (
	"fmt"
	"github.com/gfa-inc/gfa/common/validatorx"
	"github.com/go-playground/validator/v10"
)

type ParamErr struct {
	Message string
}

func (p *ParamErr) Error() string {
	return p.Message
}

func (p ParamErr) WithField(val ...any) *ParamErr {
	p.Message = fmt.Sprintf(p.Message, val...)
	return &p
}

func NewParamErr(data any) *ParamErr {
	var message string
	switch msg := data.(type) {
	case string:
		message = msg
	case validator.ValidationErrors:
		for _, err := range msg {
			message = err.Translate(validatorx.Trans)
			break
		}
	case error:
		message = msg.Error()
	}
	return &ParamErr{
		Message: message,
	}
}

type BizErr struct {
	Code    string
	Message string
}

func (b *BizErr) Error() string {
	return b.Message
}

func (b BizErr) WithField(vals ...any) *BizErr {
	b.Message = fmt.Sprintf(b.Message, vals...)
	return &b
}

func NewBizErr(code, message string) *BizErr {
	return &BizErr{
		Code:    code,
		Message: message,
	}
}

type AuthErr struct {
	Code    string
	Message string
}

func (a *AuthErr) Error() string {
	return a.Message
}

func NewAuthErr(code, message string) *AuthErr {
	return &AuthErr{
		Code:    code,
		Message: message,
	}
}
