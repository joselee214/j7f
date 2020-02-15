/*
 *
 * Copyright 2017 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package log

import (
	"log"
	"os"
)

// Logger does underlying logging work for grpclog.
type Logger interface {
	// Info logs to INFO log. Arguments are handled in the manner of fmt.Print.
	Info(args ...interface{})
	// Infoln logs to INFO log. Arguments are handled in the manner of fmt.Println.
	Infoln(args ...interface{})
	// Infof logs to INFO log. Arguments are handled in the manner of fmt.Printf.
	Infof(format string, args ...interface{})
	// Warning logs to WARNING log. Arguments are handled in the manner of fmt.Print.
	Warning(args ...interface{})
	// Warningln logs to WARNING log. Arguments are handled in the manner of fmt.Println.
	Warningln(args ...interface{})
	// Warningf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
	Warningf(format string, args ...interface{})
	// Error logs to ERROR log. Arguments are handled in the manner of fmt.Print.
	Error(args ...interface{})
	// Errorln logs to ERROR log. Arguments are handled in the manner of fmt.Println.
	Errorln(args ...interface{})
	// Errorf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
	Errorf(format string, args ...interface{})
	// Fatal logs to ERROR log. Arguments are handled in the manner of fmt.Print.
	// gRPC ensures that all Fatal logs will exit with os.Exit(1).
	// Implementations may also call os.Exit() with a non-zero exit code.
	Fatal(args ...interface{})
	// Fatalln logs to ERROR log. Arguments are handled in the manner of fmt.Println.
	// gRPC ensures that all Fatal logs will exit with os.Exit(1).
	// Implementations may also call os.Exit() with a non-zero exit code.
	Fatalln(args ...interface{})
	// Fatalf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
	// gRPC ensures that all Fatal logs will exit with os.Exit(1).
	// Implementations may also call os.Exit() with a non-zero exit code.
	Fatalf(format string, args ...interface{})
	// V reports whether verbosity level l is at least the requested verbose level.
	V(l int) bool
}

type LoggerD struct {
	l *log.Logger
}

func NewLoggerDefault() *LoggerD {
	return &LoggerD{
		l: log.New(os.Stderr, "", log.LstdFlags),
	}
}

func (g *LoggerD) Info(args ...interface{}) {
	g.l.Print(args...)
}

func (g *LoggerD) Infoln(args ...interface{}) {
	g.l.Println(args...)
}

func (g *LoggerD) Infof(format string, args ...interface{}) {
	g.l.Printf(format, args...)
}

func (g *LoggerD) Warning(args ...interface{}) {
	g.l.Print(args...)
}

func (g *LoggerD) Warningln(args ...interface{}) {
	g.l.Println(args...)
}

func (g *LoggerD) Warningf(format string, args ...interface{}) {
	g.l.Printf(format, args...)
}

func (g *LoggerD) Error(args ...interface{}) {
	g.l.Print(args...)
}

func (g *LoggerD) Errorln(args ...interface{}) {
	g.l.Println(args...)
}

func (g *LoggerD) Errorf(format string, args ...interface{}) {
	g.l.Printf(format, args...)
}

func (g *LoggerD) Fatal(args ...interface{}) {
	g.l.Fatal(args...)
	// No need to call os.Exit() again because log.Logger.Fatal() calls os.Exit().
}

func (g *LoggerD) Fatalln(args ...interface{}) {
	g.l.Fatalln(args...)
	// No need to call os.Exit() again because log.Logger.Fatal() calls os.Exit().
}

func (g *LoggerD) Fatalf(format string, args ...interface{}) {
	g.l.Fatalf(format, args...)
	// No need to call os.Exit() again because log.Logger.Fatal() calls os.Exit().
}

func (g *LoggerD) V(l int) bool {
	return l <= 0
}
