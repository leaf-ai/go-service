// Copyright 2018-2023 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package log // import "github.com/karlmutch/go-service/pkg/log"

// This file contains the implementation of a logger that adorns the logxi package with
// some common information not by default supplied by the generic code

import (
	"os"
	"sync"

	"github.com/go-stack/stack"
	logxi "github.com/karlmutch/logxi/v1"
)

var (
	hostName string
)

func init() {
	hostName, _ = os.Hostname()
}

type Logger interface {
	Label(key string, value string)
	IncludeStack(included bool) (log Logger)
	HostName(hostName string) (log Logger)

	Trace(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{}) error
	Error(msg string, args ...interface{}) error
	Fatal(msg string, args ...interface{})

	SetLevel(lvl int)

	IsDebug() bool
}

// Logger encapsulates the logging device that is used to emit logs and
// as a receiver that has the logging methods
type LoggerService struct {
	log        logxi.Logger      // The base implementation that is being encapsulated
	debugStack bool              // Should a debug stack be produced with the message
	labels     []string          // Values appended to the logging output
	included   map[string]string // The named values already in the labels
	sync.Mutex
}

// NewLogger can be used to instantiate a wrapper logger with a module label with
// output going stdout
func NewLogger(component string) (log *LoggerService) {
	logxi.DisableCallstack()

	log = &LoggerService{
		log:        logxi.New(component),
		labels:     []string{},
		included:   map[string]string{},
		debugStack: true,
	}
	if len(hostName) != 0 {
		log.labels = append(log.labels, "hostName")
		log.labels = append(log.labels, hostName)
		log.included["hostName"] = hostName
	}
	return log
}

// NewErrLogger can be used to instantiate a wrapper logger with a module label with
// output going stderr
func NewErrLogger(component string) (log *LoggerService) {
	logxi.DisableCallstack()

	log = &LoggerService{
		log:        logxi.NewLogger(logxi.NewConcurrentWriter(os.Stderr), component),
		labels:     []string{},
		included:   map[string]string{},
		debugStack: true,
	}
	if len(hostName) != 0 {
		log.labels = append(log.labels, "hostName")
		log.labels = append(log.labels, hostName)
		log.included["hostName"] = hostName
	}
	return log
}

func (l *LoggerService) Label(key string, value string) {

	// Dont allow zero length keys into the set of labels
	if len(key) == 0 {
		return
	}

	l.Lock()
	defer l.Unlock()

	// Recompute the array of items to add to logging lines during
	// label maintenance to reduce overhead
	if v, isPresent := l.included[key]; isPresent {
		// Nothing changes if the value is the same
		if v == value {
			return
		}

		// Do the change and then recompute the labels array
		if len(value) == 0 {
			delete(l.included, key)
		} else {
			l.included[key] = value
		}
		l.labels = make([]string, 0, len(l.included)*2)
		for k, v := range l.included {
			l.labels = append(l.labels, k)
			l.labels = append(l.labels, v)
		}
	} else {
		// Item was not already in the labels so just append, but only if there is a value
		if len(value) == 0 {
			return
		}
		l.included[key] = value
		l.labels = append(l.labels, key)
		l.labels = append(l.labels, value)
	}
}

// IncludeStack is used to enable a small function call stack to be included with messages
func (l *LoggerService) IncludeStack(included bool) (log Logger) {
	l.Lock()
	defer l.Unlock()

	l.debugStack = included
	return l
}

// HostName is used to add an optional host name to messages, if empty then the host name will not be output
func (l *LoggerService) HostName(hostName string) (log Logger) {
	l.Label("hostName", hostName)
	return l
}

// Trace is a method for output of trace level messages
// with a varargs style list of parameters that is formatted
// as label and then the value in a single list
func (l *LoggerService) Trace(msg string, args ...interface{}) {
	if !l.IsTrace() {
		return
	}

	allArgs := append([]interface{}{}, args...)

	l.Lock()
	defer l.Unlock()

	if l.debugStack {
		allArgs = append(allArgs, "stack")
		allArgs = append(allArgs, stack.Trace()[1:].TrimRuntime())
	}

	for _, label := range l.labels {
		allArgs = append(allArgs, label)
	}

	l.log.Trace(msg, allArgs...)
}

// Debug is a method for output of debugging level messages
// with a varargs style list of parameters that is formatted
// as label and then the value in a single list
func (l *LoggerService) Debug(msg string, args ...interface{}) {
	if !l.IsDebug() {
		return
	}

	allArgs := append([]interface{}{}, args...)

	l.Lock()
	defer l.Unlock()

	if l.debugStack {
		allArgs = append(allArgs, "stack")
		allArgs = append(allArgs, stack.Trace()[1:].TrimRuntime())
	}

	for _, label := range l.labels {
		allArgs = append(allArgs, label)
	}

	l.log.Debug(msg, allArgs...)
}

// Info is a method for output of informational level messages
// with a varargs style list of parameters that is formatted
// as label and then the value in a single list
func (l *LoggerService) Info(msg string, args ...interface{}) {
	if !l.IsInfo() {
		return
	}

	allArgs := append([]interface{}{}, args...)

	l.Lock()
	defer l.Unlock()

	if l.debugStack {
		allArgs = append(allArgs, "stack")
		allArgs = append(allArgs, stack.Trace()[1:].TrimRuntime())
	}

	for _, label := range l.labels {
		allArgs = append(allArgs, label)
	}

	l.log.Info(msg, allArgs...)
}

// Warn is a method for output of warning level messages
// with a varargs style list of parameters that is formatted
// as label and then the value in a single list
func (l *LoggerService) Warn(msg string, args ...interface{}) error {
	if !l.IsWarn() {
		return nil
	}

	allArgs := append([]interface{}{}, args...)

	l.Lock()
	defer l.Unlock()

	if l.debugStack {
		allArgs = append(allArgs, "stack")
		allArgs = append(allArgs, stack.Trace()[1:].TrimRuntime())
	}

	for _, label := range l.labels {
		allArgs = append(allArgs, label)
	}

	return l.log.Warn(msg, allArgs...)
}

// Error is a method for output of error level messages
// with a varargs style list of parameters that is formatted
// as label and then the value in a single list
func (l *LoggerService) Error(msg string, args ...interface{}) error {

	allArgs := append([]interface{}{}, args...)

	l.Lock()
	defer l.Unlock()

	allArgs = append(allArgs, "stack")
	allArgs = append(allArgs, stack.Trace()[1:].TrimRuntime())

	for _, label := range l.labels {
		allArgs = append(allArgs, label)
	}

	return l.log.Error(msg, allArgs...)
}

// Fatal is a method for output of fatal level messages
// with a varargs style list of parameters that is formatted
// as label and then the value in a single list
func (l *LoggerService) Fatal(msg string, args ...interface{}) {
	allArgs := append([]interface{}{}, args...)

	l.Lock()
	defer l.Unlock()

	allArgs = append(allArgs, "stack")
	allArgs = append(allArgs, stack.Trace()[1:].TrimRuntime())

	for _, label := range l.labels {
		allArgs = append(allArgs, label)
	}

	l.log.Fatal(msg, allArgs...)
}

// Log is a method for output of parameterized level messages
// with a varargs style list of parameters that is formatted
// as label and then the value in a single list
func (l *LoggerService) Log(level int, msg string, args []interface{}) {
	allArgs := append([]interface{}{}, args...)

	l.Lock()
	defer l.Unlock()

	if level < logxi.LevelWarn {
		allArgs = append(allArgs, "stack")
		allArgs = append(allArgs, stack.Trace()[1:].TrimRuntime())
	}

	for _, label := range l.labels {
		allArgs = append(allArgs, label)
	}

	l.log.Log(level, msg, allArgs)
}

// SetLevel can be used to set the threshold for the level of messages
// that will be output by the logger
func (l *LoggerService) SetLevel(lvl int) {
	l.Lock()
	defer l.Unlock()
	l.log.SetLevel(lvl)
}

// IsTrace returns true in the event that the theshold logging level
// allows for trace messages to appear in the output
func (l *LoggerService) IsTrace() bool {
	l.Lock()
	defer l.Unlock()
	return l.log.IsTrace()
}

// IsDebug returns true in the event that the theshold logging level
// allows for debugging messages to appear in the output
func (l *LoggerService) IsDebug() bool {
	l.Lock()
	defer l.Unlock()
	return l.log.IsDebug()
}

// IsInfo returns true in the event that the theshold logging level
// allows for informational messages to appear in the output
func (l *LoggerService) IsInfo() bool {
	l.Lock()
	defer l.Unlock()
	return l.log.IsInfo()
}

// IsWarn returns true in the event that the theshold logging level
// allows for warning messages to appear in the output
func (l *LoggerService) IsWarn() bool {
	l.Lock()
	defer l.Unlock()
	return l.log.IsWarn()
}
