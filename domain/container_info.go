package domain

import (
	"net"
	"time"
)

type ContainerSortProperty string

const (
	ContainerSortByIP          ContainerSortProperty = "ip"
	ContainerSortByLastPing                          = "last_ping"
	ContainerSortByLastSuccess                       = "last_success"
)

type ContainerInfo struct {
	IP          net.IPAddr `json:"ip"`
	LastPing    time.Time  `json:"last_ping"`
	LastSuccess time.Time  `json:"last_success"`
}
