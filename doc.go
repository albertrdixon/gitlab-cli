package main

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	version      = "dev"
	commit, date string
)

func shortVersion() string { return version }
func fullVersion() string {
	return fmt.Sprintf(
		`%s (%s on %s/%s; %s)`,
		strings.Join([]string{version, commit, date}, "/"),
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Compiler,
	)
}
