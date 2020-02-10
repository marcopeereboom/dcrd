// Copyright (c) 2013-2014 The btcsuite developers
// Copyright (c) 2015-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/decred/slog"
)

// log is a logger that is initialized with no output filters.  This
// means the package will not perform any logging by default until the caller
// requests it.
// The default amount of logging is none.
var (
	log     = slog.Disabled
	trsyLog = slog.Disabled
)

// UseLogger uses a specified Logger to output package logging info.
func UseLogger(logger slog.Logger) {
	log = logger
}

// UseTreasuryLogger uses a specified Logger to output treasury logging info.
func UseTreasuryLogger(logger slog.Logger) {
	trsyLog = logger
}
