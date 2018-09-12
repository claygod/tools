package logger

// Logger
// Main
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"bytes"
	"fmt"
	"io"
)

/*
Logger - logging with context and branching support
*/
type Logger struct {
	writer  io.Writer
	parent  *Logger
	title   string
	context interface{}
}

/*
New - create new Logger
*/
func New(w io.Writer) *Logger {
	return &Logger{writer: w}
}

/*
Error - write an `error` message
*/
func (l *Logger) Error(ctx interface{}) *Logger {
	return &Logger{
		parent:  l,
		title:   "Error",
		context: ctx,
	}
}

/*
Warning - write an `warning` message
*/
func (l *Logger) Warning(ctx interface{}) *Logger {
	return &Logger{
		parent:  l,
		title:   "Warning",
		context: ctx,
	}
}

/*
Info - write down an information message
*/
func (l *Logger) Info(ctx interface{}) *Logger {
	return &Logger{
		parent:  l,
		title:   "Info",
		context: ctx,
	}
}

/*
Context - add context to the log
*/
func (l *Logger) Context(title string, ctx interface{}) *Logger {
	nl := &Logger{
		parent:  l,
		title:   title,
		context: ctx,
	}
	return nl
}

/*
Send - send information to the log
*/
func (l *Logger) Send() (int, error) {
	buf := bytes.NewBuffer([]byte{})
	curLogger := l
	bufStr := make([]string, 0)
	for {
		if curLogger.writer != nil {
			break
		}
		bufStr = append(bufStr, fmt.Sprintf("%s: %v. ", curLogger.title, curLogger.context))
		curLogger = curLogger.parent
	}
	for i := len(bufStr) - 1; i >= 0; i-- {
		if count, err := buf.WriteString(bufStr[i]); err != nil {
			return count, err
		}
	}
	return curLogger.writer.Write(buf.Bytes())
}
