// Copyright 2018-2022 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package runtime // import "github.com/karlmutch/go-service/pkg/runtime"

// This file contains functions to handle the Go language build information

import (
	"fmt"
	"runtime/debug"
	"strconv"
)

type BuildMetadata struct {
	Revision      string
	ShortRevision string
	Time          string
	Dirty         string

	Arch string
	OS   string
}

var (
	BuildInfo = BuildMetadata{}
)

func init() {

	if info, isOK := debug.ReadBuildInfo(); isOK && info != nil {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				BuildInfo.Revision = setting.Value[:]
			case "vcs.time":
				BuildInfo.Time = setting.Value[:]
			case "vcs.modified":
				if dirty, errGo := strconv.ParseBool(setting.Value[:]); errGo != nil {
					fmt.Println("could not determine if the build was using uncommitted code", "error", errGo.Error())
				} else {
					if dirty {
						BuildInfo.Dirty = "+"
					}
				}
			case "GOARCH":
				BuildInfo.Arch = setting.Value[:]
			case "GOOS":
				BuildInfo.OS = setting.Value[:]
			}
		}
	}
	if len(BuildInfo.Revision) != 0 {
		BuildInfo.ShortRevision = BuildInfo.Revision[:7] + BuildInfo.Dirty
	}
}
