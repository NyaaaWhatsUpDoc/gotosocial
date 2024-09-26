// GoToSocial
// Copyright (C) GoToSocial Authors admin@gotosocial.org
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package gtserror

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/superseriousbusiness/gotosocial/internal/log"
)

// New returns a new error,
// prepended with calling function. It
// functions similarly to errors.New().
//
//go:noinline
func New(msg string) error {
	return &cerror{
		c: log.Caller(3),
		e: errors.New(msg),
	}
}

// Newf returns a new formatted error,
// prepended with calling function. It
// functions similarly to fmt.Errorf().
//
//go:noinline
func Newf(msgf string, args ...any) error {
	return &cerror{
		c: log.Caller(3),
		e: fmt.Errorf(msgf, args...),
	}
}

// NewAt returns a new error,
// prepended with calling function. It
// functions similarly to errors.New().
//
// Calldepth allows you to specify how many
// function frames to skip in determining
// calling function name. You should manually
// check this returns expected function name
// before deploy, this very much depends on
// how the inliner behaves during compilation.
// Generally a value of 3 is a good start.
//
// See runtime.Callers() and runtime.FuncForPC()
// for more information on calldepth usage.
//
//go:noinline
func NewAt(calldepth int, msg string) error {
	return &cerror{
		c: log.Caller(calldepth + 1),
		e: errors.New(msg),
	}
}

// NewfAt returns a new formatted error,
// prepended with calling function. It
// functions similarly to fmt.Errorf().
//
// Calldepth allows you to specify how many
// function frames to skip in determining
// calling function name. You should manually
// check this returns expected function name
// before deploy, this very much depends on
// how the inliner behaves during compilation.
// Generally a value of 3 is a good start.
//
// See runtime.Callers() and runtime.FuncForPC()
// for more information on calldepth usage.
//
//go:noinline
func NewfAt(calldepth int, msgf string, args ...any) error {
	return &cerror{
		c: log.Caller(calldepth + 1),
		e: fmt.Errorf(msgf, args...),
	}
}

// Wrap returns a new wrapped error,
// prepended with calling function.
//
//go:noinline
func Wrap(err error) error {
	return &cerror{
		c: log.Caller(3),
		e: err,
	}
}

// WrapAt returns a new wrapped error,
// prepended with calling function.
//
// Calldepth allows you to specify how many
// function frames to skip in determining
// calling function name. You should manually
// check this returns expected function name
// before deploy, this very much depends on
// how the inliner behaves during compilation.
// Generally a value of 3 is a good start.
//
// See runtime.Callers() and runtime.FuncForPC()
// for more information on calldepth usage.
//
//go:noinline
func WrapAt(calldepth int, err error) error {
	return &cerror{
		c: log.Caller(calldepth + 1),
		e: err,
	}
}

// NewResponseError crafts an error from provided HTTP response
// including the method, status and body (if any provided). This
// will also wrap the returned error using WithStatusCode() and
// will include the caller function name as a prefix.
//
//go:noinline
func NewFromResponse(rsp *http.Response) error {
	// Build error with message without
	// using "fmt", as chances are this will
	// be used in a hot code path and we
	// know all the incoming types involved.
	err := NewAt(3, ""+
		rsp.Request.Method+
		" request to "+
		rsp.Request.URL.String()+
		" failed: status=\""+
		rsp.Status+
		"\" body=\""+
		drainBody(rsp.Body, 256)+
		"\"",
	)

	// Wrap error to provide status code.
	return WithStatusCode(err, rsp.StatusCode)
}

// cerror wraps an error with a string
// prefix of the caller function name.
type cerror struct {
	c string
	e error
}

func (ce *cerror) Error() string {
	msg := ce.e.Error()
	return ce.c + ": " + msg
}

func (ce *cerror) Unwrap() error {
	return ce.e
}
