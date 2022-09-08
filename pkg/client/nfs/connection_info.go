package nfs

import (
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NFSConnectionInfo class
type NFSConnectionInfo struct {
	Hostname string
	Port     int
	Path     string
}

func getConnectionInfoFromMap(params map[string]string, connInfo *NFSConnectionInfo) error {
	for k, v := range params {
		switch k {
		case "host":
			connInfo.Hostname = v
		case "port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %s", k, err)
			}
			connInfo.Port = p
		case "path":
			connInfo.Path = v
		default:
			// ignore
		}
	}

	return nil
}

// GetConnectionInfo returns NFSConnectionInfo value from param map
func GetConnectionInfo(configs map[string]string) (*NFSConnectionInfo, error) {
	connInfo := NFSConnectionInfo{}

	err := getConnectionInfoFromMap(configs, &connInfo)
	if err != nil {
		return nil, err
	}

	if len(connInfo.Hostname) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument host is empty")
	}

	if len(connInfo.Path) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument path is empty")
	}

	if connInfo.Port <= 0 {
		// default
		connInfo.Port = 2049
	}

	return &connInfo, nil
}
