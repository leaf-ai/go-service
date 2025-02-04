// Copyright 2018-2022 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package server // import "github.com/leaf-ai/go-service/pkg/server"

// This file contains functions related to Kubernetes (k8s) support for pod based servers.
// The server can use k8s to watch and load ConfigMap information that it can use
// to manage its life cycle and in the future to load configuration information.
//
// The choice to make use of the package from Eric Chiang is driven by the
// package dependency issues with using the official go client.  It rivals
// the spagetti dependencies of Dockers buildkit, borderline horrific.  The chosen
// package has a single dependency and trades off using generated protobuf structures
// and so it wired to the k8s versions via that method, a tradeoff I'm willing to
// make based on my attempts with BuildKit.

import (
	"context"
	"github.com/leaf-ai/go-service/pkg/log"
	"sync"
	"time"

	"github.com/go-stack/stack"
	"github.com/lthibault/jitterbug"

	"github.com/jjeffery/kv" // MIT License

	"github.com/leaf-ai/go-service/pkg/types"
)

var (
	k8sClient *interface{}
	//k8sClient  *k8s.Client
	k8sInitErr kv.Error

	_ = attemptK8sStart()

	protect sync.Mutex
)

func attemptK8sStart() (err kv.Error) {
	protect.Lock()
	defer protect.Unlock()

	return kv.NewError("Kubernetes start is not implemented")
	//
	//if client, errGo := k8s.NewInClusterClient(); errGo != nil {
	//	k8sInitErr = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	//} else {
	//	k8sClient = client
	//}
	//
	//return k8sInitErr
}

//func watchCMaps(ctx context.Context, namespace string) (cmChange chan *core.ConfigMap, err kv.Error) {
//
//	configMap := core.ConfigMap{}
//	watcher, errGo := k8sClient.Watch(ctx, namespace, &configMap)
//	if errGo != nil {
//		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
//	}
//
//	cmChange = make(chan *core.ConfigMap, 1)
//	go func() {
//
//		defer func() {
//			if watcher != nil {
//				watcher.Close() // Always close the returned watcher.
//			}
//		}()
//
//		for {
//			cm := &core.ConfigMap{}
//			// Next does not support cancellation and is blocking so we have to
//			// abandon this thread and simply let it run unmanaged
//			_, err := watcher.Next(cm)
//			if err != nil {
//				select {
//				case <-ctx.Done():
//					return
//				default:
//				}
//				if watcher != nil {
//					watcher.Close()
//					watcher = nil
//				}
//				// watcher encountered and error, create a new watcher
//				watcher, _ = k8sClient.Watch(ctx, namespace, &configMap)
//				continue
//			}
//			select {
//			case cmChange <- cm:
//			case <-time.After(time.Second):
//				spew.Dump(*cm)
//			}
//		}
//	}()
//	return cmChange, nil
//}

// MonitorK8s is used to send appropriate errors into an error reporting channel
// on a regular basis if the k8s connectivity state changes
//
func MonitorK8s(ctx context.Context, errC chan<- kv.Error) {

	t := jitterbug.New(time.Second*30, &jitterbug.Norm{Stdev: time.Second * 3})
	defer t.Stop()

	k8sLastErr := kv.NewError("")
	for {
		select {
		case <-ctx.Done():
			return

		case <-t.C:
			func() {
				attemptRestart := false
				protect.Lock()

				if k8sClient == nil {
					if k8sInitErr != nil && k8sInitErr != k8sLastErr {
						// Attempt to send the error back to any listener and timeout after a second.
						// The jitter based timer will provide another opportunity in 30 or thereabouts
						// seconds should this timeout
						select {
						case errC <- k8sInitErr:
							k8sLastErr = k8sInitErr
						case <-time.After(time.Second):
						}
					}
					attemptRestart = true

				}
				protect.Unlock()

				if attemptRestart {
					// Doing this re-applies the lock so we do it outside of the earlier block
					attemptK8sStart()
				}
			}()
		}
	}
}

// IsAliveK8s is used to extract any kv.in the state of the k8s client api connection.
//
// A nil returned indicates k8s is working and in use, otherwise a descriptive error
// is returned.
//
func IsAliveK8s() (err kv.Error) {
	protect.Lock()
	defer protect.Unlock()

	if k8sClient == nil {
		k8sInitErr = kv.NewError("Kubernetes uninitialized or no cluster present").With("stack", stack.Trace().TrimRuntime())
	}
	return k8sInitErr
}

// ConfigK8s is used to pull the values from a named config map in k8s
//
// This function will return an empty map and and error value on failure.
//
//func ConfigK8s(ctx context.Context, namespace string, name string) (values map[string]string, err kv.Error) {
//	values = map[string]string{}
//
//	if err = IsAliveK8s(); err != nil {
//		return values, nil
//	}
//	cfg := &core.ConfigMap{}
//
//	if errGo := k8sClient.Get(ctx, namespace, name, cfg); errGo != nil {
//		return values, kv.Wrap(errGo).With("namespace", namespace).With("name", name).With("stack", stack.Trace().TrimRuntime())
//	}
//
//	if name == *cfg.Metadata.Name {
//		fmt.Println(spew.Sdump(cfg.Data), stack.Trace().TrimRuntime())
//		return cfg.Data, nil
//	}
//	return values, kv.NewError("configMap not found").With("namespace", namespace).With("name", name).With("stack", stack.Trace().TrimRuntime())
//}

// K8sStateUpdate encapsulates the known kubernetes state within which the server finds itself.
//
type K8sStateUpdate struct {
	Name  string
	State types.K8sState
}

// K8sConfigUpdate encapsulates an update for some config map in the scope of server run-time environment.
//
type K8sConfigUpdate struct {
	NameSpace string
	Name      string
	State     map[string]string
}

// ListenK8s will register a listener to watch for pod specific configMaps in k8s
// and will relay state changes to a channel,  the global state map should exist
// at the bare minimum.  A state change in either map superseded any previous
// state.
//
// This is a blocking function that will return either upon an error in API calls
// to the cluster API or when the ctx is Done().
//
//func ListenK8s(ctx context.Context, namespace string, globalMap string, podMap string, updateC chan<- K8sStateUpdate, errC chan<- kv.Error, logger *log.Logger) (err kv.Error) {
//
//	// If k8s is not being used ignore this feature
//	if err = IsAliveK8s(); err != nil {
//		return err
//	}
//
//	// Starts the application level state watching
//	currentState := K8sStateUpdate{
//		State: types.K8sUnknown,
//	}
//
//	// Start the k8s configMap watcher
//	cmChanges, err := watchCMaps(ctx, namespace)
//	if err != nil {
//		// The implication of an error here is that we will never get updates from k8s
//		return err
//	}
//
//	fmt.Println("k8s watcher starting in namespace", namespace)
//	defer fmt.Println("k8s watcher stopping")
//
//	// Once every 3 minutes for so we will force the state propagation
//	// to ensure that modules started after this module has started see something
//	refresh := jitterbug.New(time.Minute*3, &jitterbug.Norm{Stdev: time.Second * 15})
//	defer refresh.Stop()
//
//	for {
//		select {
//		case <-ctx.Done():
//			return nil
//		case <-refresh.C:
//			// Try resending an existing state to listeners to refresh things
//			select {
//			case updateC <- currentState:
//			case <-time.After(2 * time.Second):
//			}
//		case cm := <-cmChanges:
//			if cm == nil {
//				fmt.Println("k8s watcher channel closed", namespace)
//				return
//			}
//
//			if logger != nil {
//				logger.Debug("DETECTED ConfigMap update: ", "configMap: ", *cm)
//			}
//
//			if *cm.Metadata.Namespace == namespace && (*cm.Metadata.Name == globalMap || *cm.Metadata.Name == podMap) {
//				currentState = processState(cm, currentState, updateC, errC)
//			}
//		}
//	}
//}

// ListenK8sConfigMaps will register a listener to watch for k8s config maps updates in a specified namespace
// and will relay these changes to a channel.
//
// This is a blocking function that will return either upon an error in API calls
// to the cluster API or when the ctx is Done().
//
func ListenK8sConfigMaps(ctx context.Context, namespace string, updateC chan<- K8sConfigUpdate, errC chan<- kv.Error, logger *log.Logger) (err kv.Error) {

	// If k8s is not being used ignore this feature
	if err = IsAliveK8s(); err != nil {
		return err
	}
	return nil

	//hasUpdate := false
	//lastConfigUpdate := K8sConfigUpdate{}
	//
	//// Start the k8s configMap watcher
	//cmChanges, err := watchCMaps(ctx, namespace)
	//if err != nil {
	//	// The implication of an error here is that we will never get updates from k8s
	//	return err
	//}
	//
	//logger.Info("k8s config watcher starting in namespace", namespace)
	//defer logger.Info("k8s config watcher stopping for namespace", namespace)
	//
	//// Once every 2 minutes or so we will force the state propagation
	//// to ensure that modules started after this module has started see something
	//refresh := jitterbug.New(time.Minute*2, &jitterbug.Norm{Stdev: time.Second * 15})
	//defer refresh.Stop()
	//
	//for {
	//	select {
	//	case <-ctx.Done():
	//		return nil
	//	case <-refresh.C:
	//		// Try resending the latest update to listeners to refresh things
	//		if hasUpdate {
	//			select {
	//			case updateC <- lastConfigUpdate:
	//			case <-time.After(2 * time.Second):
	//			}
	//		}
	//	case cm := <-cmChanges:
	//		if cm == nil {
	//			logger.Info("k8s config watcher channel closed", namespace)
	//			return
	//		}
	//		logger.Debug("RECEIVED ConfigMap update: ", "configMap: ", *cm)
	//		lastConfigUpdate = K8sConfigUpdate{
	//			NameSpace: namespace,
	//			Name:      *cm.Metadata.Name,
	//			State:     cm.Data,
	//		}
	//		hasUpdate = true
	//		select {
	//		case updateC <- lastConfigUpdate:
	//		case <-time.After(2 * time.Second):
	//		}
	//	}
	//}
}

//func processState(cm *core.ConfigMap, currentState K8sStateUpdate, updateC chan<- K8sStateUpdate, errC chan<- kv.Error) (newState K8sStateUpdate) {
//	if state := cm.Data["STATE"]; len(state) != 0 {
//		newState, errGo := types.K8sStateString(state)
//		if errGo != nil {
//			msg := kv.Wrap(errGo).With("namespace", *cm.Metadata.Namespace).With("config", *cm.Metadata.Name).With("state", state).With("stack", stack.Trace().TrimRuntime())
//			select {
//			case errC <- msg:
//			case <-time.After(2 * time.Second):
//				fmt.Println(errGo)
//			}
//		}
//		if newState == currentState.State && *cm.Metadata.Name == currentState.Name {
//			return currentState
//		}
//		update := K8sStateUpdate{
//			Name:  *cm.Metadata.Name,
//			State: newState,
//		}
//		// Try sending the new state to listeners within the server invoking this function
//		select {
//		case updateC <- update:
//			currentState = update
//		case <-time.After(2 * time.Second):
//			// If the message could not be sent try to wakeup the error logger
//			msg := kv.NewError("could not update state").With("namespace", *cm.Metadata.Namespace).With("config", *cm.Metadata.Name).With("state", state).With("stack", stack.Trace().TrimRuntime())
//			select {
//			case errC <- msg:
//			case <-time.After(2 * time.Second):
//				fmt.Println(msg)
//			}
//		}
//	}
//	return currentState
//}
