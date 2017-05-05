package global

import (
	"fmt"
)

// Version 控制版本号
func Version() string {
	return fmt.Sprintf("serviceGen v%d.%d.%d", versionMajor, versionMinor, versionPatch)
}

const (
	versionMajor = 0
	versionMinor = 0
	versionPatch = 1
)
