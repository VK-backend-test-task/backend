package domain

import "net"

type ContainerInfo struct {
	IP net.IPAddr `json:"ip"`
}
