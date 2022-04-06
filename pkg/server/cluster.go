// Copyright 2018-2022 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package server // import "github.com/leaf-ai/go-service/pkg/server"

import (
	"context"
	"time"

	"github.com/andreidenissov-cog/go-service/pkg/log"

	"github.com/jjeffery/kv" // MIT License
)

var (
	configListeners *ConfigListeners
)

func K8sConfigUpdates() (l *ConfigListeners) {
	return configListeners
}

// initiateK8s runs until either ctx is Done or the listener is running successfully
func InitiateK8s(ctx context.Context, namespace string, cfgMap string, readyC chan struct{}, staleMsg time.Duration, logger *log.Logger, errorC chan kv.Error) {

	// If the user did specify the k8s parameters then we need to process the k8s configs
	if len(namespace) == 0 || len(cfgMap) == 0 {
		return
	}

	configListeners = NewConfigBroadcast(ctx, errorC)

	func() {
		defer recover()
		close(readyC)
	}()

	// Watch for k8s API connectivity events that are of interest and use the errorC to surface them
	go MonitorK8s(ctx, errorC)

	// The convention exists that the per machine configmap name is simply the hostname
	//podMap := os.Getenv("HOSTNAME")

	// In the event that initializing the k8s listener fails we try once every 30 seconds to get it working
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			// If k8s is specified we need to start a listener for config maps updates:
			if err := ListenK8sConfigMaps(ctx, namespace, configListeners.Master, errorC, logger); err != nil {
				logger.Warn("k8s config maps monitoring offline", "error", err.Error())
			}
		case <-ctx.Done():
			return
		}
	}
}
