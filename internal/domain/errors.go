package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound       = errors.New("资源不存在")
	ErrConflict       = errors.New("数据已被其他操作更新")
	ErrInvalidState   = errors.New("当前状态不允许此操作")
	ErrValidation     = errors.New("字段校验失败")
	ErrRoleSeparation = errors.New("提交者不能完成最终安全评审")
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	Fields []FieldError `json:"fields"`
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return ErrValidation.Error()
	}
	return fmt.Sprintf("%s：%s %s", ErrValidation, e.Fields[0].Field, e.Fields[0].Message)
}

func (e *ValidationError) Unwrap() error { return ErrValidation }

func AddFieldError(fields []FieldError, field, message string) []FieldError {
	return append(fields, FieldError{Field: field, Message: message})
}
