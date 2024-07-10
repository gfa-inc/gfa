package core

type ParamErr struct {
	Message string
}

func (p *ParamErr) Error() string {
	return p.Message
}

func NewParamErr(data any) *ParamErr {
	var message string
	switch msg := data.(type) {
	case string:
		message = msg
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

func NewBizErr(code, message string) *BizErr {
	return &BizErr{
		Code:    code,
		Message: message,
	}
}
