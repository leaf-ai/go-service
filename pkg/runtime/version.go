// Copyright 2018-2022 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package runtime // import "github.com/karlmutch/go-service/pkg/runtime"

// This file contains functions to handle the Go language build information

import (
	"fmt"
	"runtime/debug"
	"strconv"
)

type BuildMetadata struct {
	Revision      string // Full commit ID or tag for this build for this present module
	ShortRevision string // Abberviated commit ID with a plus character if the build had uncommitted files
	Time          string // The time of the build, typically expressed as UTC
	Dirty         string // Dirty build, uncommitted files, character.  Contains a plus character if build was dirty or empty string if not.

	Arch string // The hardware architecture for this binary
	OS   string // The targetted operating system

	GoVersion   string // The Go compiler version being used
	ProjectPath string // The path for the project
	ModulePath  string // The git location for the modules root
}

var (
	// BuildInfo is a globally accessible build information block from the Go compiler about
	// the build environment
	BuildInfo = BuildMetadata{}
)

func init() {

	if info, isOK := debug.ReadBuildInfo(); isOK && info != nil {

		BuildInfo.GoVersion = info.GoVersion
		BuildInfo.ProjectPath = info.Path
		BuildInfo.ModulePath = info.Main.Path

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
