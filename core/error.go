package core

type ParameterError struct {
	Message string
}

func (p *ParameterError) Error() string {
	return p.Message
}

func NewParameterError(message string) *ParameterError {
	return &ParameterError{
		Message: message,
	}
}

type BizError struct {
	Code    string
	Message string
}

func (b *BizError) Error() string {
	return b.Message
}

func NewBizError(code, message string) *BizError {
	return &BizError{
		Code:    code,
		Message: message,
	}
}
