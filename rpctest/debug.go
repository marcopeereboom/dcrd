// Copyright (c) 2016-2017 The btcsuite developers
// Copyright (c) 2017-2020 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpctest

import "testing"

// This package is very hard to debug so we add a couple of variables that
// enable debug and tracing output. Leave them false before committing to
// master.
var (
	debug bool // Set to true to enable additional verbosity.
	trace bool // Set to true to enable tracing.
)

func init() {
	debug = true
	trace = true
}

func tracef(t *testing.T, format string, args ...interface{}) {
	if !trace {
		return
	}
	t.Logf(format, args...)
}

func debugf(t *testing.T, format string, args ...interface{}) {
	if !debug {
		return
	}
	t.Logf(format, args...)
}
