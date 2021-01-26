// Copyright 2018-2021 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"

	"github.com/leaf-ai/go-service/internal/types"
	"github.com/leaf-ai/go-service/pkg/server"

	"github.com/go-stack/stack"
	"github.com/jjeffery/kv" // MIT License
)

// This file contains the implementation of a test that will simulate a state change
// for the server and will verify that the schedulers respond appropriately. States
// are controlled using kubernetes and so this test will exercise the state management
// without using the k8s modules, these are tested separately

// TestBroadcast tests the fan-out of Kubernetes state updates using Go channels. This is
// primarily a unit test when for the k8s cluster is not present
//
func TestBroadcast(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
	defer cancel()

	errorC := make(chan kv.Error, 1)

	l := server.NewStateBroadcast(ctx, errorC)

	// Create three listeners
	listeners := []chan server.K8sStateUpdate{
		make(chan server.K8sStateUpdate, 1),
		make(chan server.K8sStateUpdate, 1),
		make(chan server.K8sStateUpdate, 1),
	}
	for _, listener := range listeners {
		if _, err := l.Add(listener); err != nil {
			t.Fatal(err)
		}
	}

	failed := false
	err := kv.NewError("")
	doneC := make(chan struct{}, 1)

	go func() {
		defer close(doneC)
		// go routine the listeners, with early finish if they are receive traffic
		for _, listener := range listeners {
			select {
			case <-listener:
			case <-ctx.Done():
				err = kv.NewError("one of the listeners received no first message").With("stack", stack.Trace().TrimRuntime())
				failed = true
				return
			}
		}
		// Now check that no listener gets a second message
		for _, listener := range listeners {
			select {
			case <-listener:
				err = kv.NewError("one of the listeners received an unexpected second message").With("stack", stack.Trace().TrimRuntime())
				failed = true
				return
			case <-time.After(20 * time.Millisecond):
			}
		}
	}()

	// send something out, let it be consumed and if it is not then we have an issue
	select {
	case l.Master <- server.K8sStateUpdate{
		State: types.K8sRunning,
		Name:  xid.New().String(),
	}:
	case <-ctx.Done():
		t.Fatal(kv.NewError("the master channel could not be used to send a broadcast").With("stack", stack.Trace().TrimRuntime()))
	}

	// Now wait for the receiver to do its thing
	select {
	case <-doneC:
	case <-ctx.Done():
		t.Fatal(kv.NewError("the receiver channel(s) timed out").With("stack", stack.Trace().TrimRuntime()))
	}

	// see what happened
	if failed {
		t.Fatal(err)
	}
}
