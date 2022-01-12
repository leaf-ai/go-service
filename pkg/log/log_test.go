// Copyright 2022 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package log // import "github.com/karlmutch/go-service/pkg/log"

import "testing" // This file contains the implementation of a logger that adorns the logxi package with
// some common information not by default supplied by the generic code

func TestOutput(t *testing.T) {
	logger := NewLogger("unit-test")
	logger.Info("test should have host and stack")

	logger.HostName("")
	logger.IncludeStack(false)
	logger.Info("test should NOT have host and stack")
}
