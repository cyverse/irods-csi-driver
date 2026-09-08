package commons

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var (
	driverVersion string
	gitCommit     string
	buildDate     string
)

// VersionInfo object contains version related info
type VersionInfo struct {
	DriverVersion string `json:"driverVersion"`
	GitCommit     string `json:"gitCommit"`
	BuildDate     string `json:"buildDate"`
	GoVersion     string `json:"goVersion"`
	Compiler      string `json:"compiler"`
	Platform      string `json:"platform"`
}

// GetVersion returns VersionInfo object
func GetVersion() VersionInfo {
	return VersionInfo{
		DriverVersion: driverVersion,
		GitCommit:     gitCommit,
		BuildDate:     buildDate,
		GoVersion:     runtime.Version(),
		Compiler:      runtime.Compiler,
		Platform:      fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// GetDriverVersion returns driver version in string
func GetDriverVersion() string {
	return driverVersion
}

// GetVersionJSON returns VersionInfo object in JSON string
func GetVersionJSON() (string, error) {
	info := GetVersion()
	marshalled, err := json.MarshalIndent(&info, "", "  ")
	if err != nil {
		return "", err
	}
	return string(marshalled), nil
}
