// Copyright 2020-2022 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package server // import "github.com/karlmutch/go-service/pkg/server"

import (
	"os"
	"runtime"

	"github.com/dustin/go-humanize"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

// Resources is a receiver for resource related methods used to describe machine level capabilities
//
type Resources struct{}

// FetchMachineResources extracts the current system state in terms of memory etc
// and coverts this into the resource specification used to pass machine characteristics
// around.
//
func (*Resources) FetchMachineResources() (rsc *Resource) {

	rsc = &Resource{
		Cpus:   uint(runtime.NumCPU()),
		Gpus:   0,
		GpuMem: "0",
	}

	v, _ := mem.VirtualMemory()
	rsc.Ram = humanize.Bytes(v.Free)

	if dir, errGo := os.Getwd(); errGo != nil {
		if di, errGo := disk.Usage(dir); errGo != nil {
			rsc.Hdd = humanize.Bytes(di.Free)
		}
	}

	return rsc
}
