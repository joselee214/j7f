package errors

import (
	"fmt"
	"github.com/joselee214/j7f/proto/common"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"io"
	"reflect"
)

type CommonError int32

const (
	OK      = 0
	SUCCESS = "success"
	// 初始值, 无意义
	CommonError_INIT CommonError = 0
	// 处理超时
	CommonError_PROCESSING_TIMEOUT CommonError = 10001
)

type errorCode interface {
	String() string
}

type Error struct {
	code int64
	err  string

	*stack
}

func (e *Error) ResHeader() *common.BusinessStatus {
	return &common.BusinessStatus{
		MsgCode: &common.BusinessStatus_Code{Code: int32(e.code)},
		Msg:     e.err,
	}
}

func GetResHeader(err error) *common.BusinessStatus {
	if err == nil {
		return &common.BusinessStatus{
			MsgCode: &common.BusinessStatus_Code{Code: OK},
			Msg:     SUCCESS,
		}
	}
	e, ok := err.(*Error)
	if ok {
		return e.ResHeader()
	}

	return &common.BusinessStatus{
		MsgCode: &common.BusinessStatus_Code{Code: int32(codes.Unknown)},
		Msg:     err.Error(),
	}
}

func NewFromCode(code interface{}) *Error {
	return &Error{
		code:  reflect.ValueOf(code).Int(),
		err:   code.(errorCode).String(),
		stack: callers(),
	}
}

func NewFromError(err error) *Error {
	e := &Error{
		err:   err.Error(),
		stack: callers(),
	}
	e.code = int64(codes.Unknown)

	return e
}

func New(s string) *Error {
	e := &Error{
		err:   s,
		stack: callers(),
	}
	e.code = int64(codes.Unknown)

	return e
}

func Errorf(format string, args ...interface{}) *Error {
	return &Error{
		err:   fmt.Sprintf(format, args...),
		stack: callers(),
	}
}

func (e *Error) Error() string { return e.err }

func (e *Error) String() string { return e.err }

func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, e.err)
			e.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.err)
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.err)
	}
}

func (e *Error) GRPCStatus() *status.Status {
	return &status.Status{
		Code:    int32(e.code),
		Message: e.err,
	}
}

func WithStack(err error) error {
	if err == nil {
		return nil
	}
	return &withStack{
		err,
		callers(),
	}
}

type withStack struct {
	error
	*stack
}

func (w *withStack) Cause() error { return w.error }

func (w *withStack) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "%+v", w.Cause())
			w.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, w.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", w.Error())
	}
}

// Wrap returns an error annotating err with a stack trace
// at the point Wrap is called, and the supplied message.
// If err is nil, Wrap returns nil.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	err = &withMessage{
		cause: err,
		msg:   message,
	}
	return &withStack{
		err,
		callers(),
	}
}

// Wrapf returns an error annotating err with a stack trace
// at the point Wrapf is called, and the format specifier.
// If err is nil, Wrapf returns nil.
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	err = &withMessage{
		cause: err,
		msg:   fmt.Sprintf(format, args...),
	}
	return &withStack{
		err,
		callers(),
	}
}

// WithMessage annotates err with a new message.
// If err is nil, WithMessage returns nil.
func WithMessage(err error, message string) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		cause: err,
		msg:   message,
	}
}

// WithMessagef annotates err with the format specifier.
// If err is nil, WithMessagef returns nil.
func WithMessagef(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		cause: err,
		msg:   fmt.Sprintf(format, args...),
	}
}

type withMessage struct {
	cause error
	msg   string
}

func (w *withMessage) Error() string { return w.msg + ": " + w.cause.Error() }
func (w *withMessage) Cause() error  { return w.cause }

func (w *withMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "%+v\n", w.Cause())
			_, _ = io.WriteString(s, w.msg)
			return
		}
		fallthrough
	case 's', 'q':
		_, _ = io.WriteString(s, w.Error())
	}
}

// Cause returns the underlying cause of the error, if possible.
// An error value has a cause if it implements the following
// interface:
//
//     type causer interface {
//            Cause() error
//     }
//
// If the error does not implement Cause, the original error will
// be returned. If the error is nil, nil will be returned without further
// investigation.
func Cause(err error) error {
	type causer interface {
		Cause() error
	}

	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	return err
}
