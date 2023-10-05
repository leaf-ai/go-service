// Copyright 2021-2023 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package minio_local

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-stack/stack"
	"github.com/karlmutch/envflag"
	"github.com/karlmutch/kv"
)

var (
	topDir = flag.String("top-dir", "../..", "The location of the top level source directory for locating test files")
)

func TestMain(m *testing.M) {
	// Only perform this Parsed check inside the test framework. Do not be tempted
	// to do this in the main of our production package
	//
	if !flag.Parsed() {
		envflag.Parse()
	}

	// Make sure that any test files can be found via a valid topDir argument on the CLI
	if stat, errGo := os.Stat(*topDir); os.IsNotExist(errGo) {
		fmt.Println(kv.Wrap(errGo).With("top-dir", *topDir).With("stack", stack.Trace().TrimRuntime()))
		os.Exit(-1)
	} else {
		if !stat.Mode().IsDir() {
			fmt.Println(kv.NewError("not a directory").With("top-dir", *topDir).With("stack", stack.Trace().TrimRuntime()))
			os.Exit(-1)
		}

	}
	if dir, errGo := filepath.Abs(*topDir); errGo != nil {
		fmt.Println((kv.Wrap(errGo).With("top-dir", *topDir).With("stack", stack.Trace().TrimRuntime())))
	} else {
		if errGo := flag.Set("top-dir", dir); errGo != nil {
			fmt.Println((kv.Wrap(errGo).With("top-dir", *topDir).With("stack", stack.Trace().TrimRuntime())))
		}
	}
	m.Run()
}

func TestMinioLifecycle(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	defer cancel()

	// Start the minio test server and wait for its context to timeout
	mts, errC := InitTestingMinio(ctx, false)

	func() {
		started := false

		for {
			select {
			case <-ctx.Done():
				// Give the server a second to die
				<-time.After(time.Second)

				return
			case err := <-errC:
				if err != nil {
					t.Fatal(err.Error())
				}

			case <-time.After(time.Second):
				if !started {
					if mts.ProcessState == nil {
						break
					}
					slog.DebugContext(ctx, "minio process state is available", "stack", stack.Trace().TrimRuntime())
					started = true
				}
			}
		}
	}()

	// Check to ensure the process was shutdown successfully
	if mts.ProcessState == nil {
		t.Fatal("The minio test servers process was not accessible")
	}

	<-time.After(time.Second)

	if !mts.ProcessState.Exited() {
		if mts.ProcessState.String() != "signal: killed" {
			t.Fatal("The minio test servers process has not exited", "state", mts.ProcessState.String())
		}
	}
}
