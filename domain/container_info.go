package domain

import (
	"net/netip"
	"time"
)

type ContainerSortProperty string

const (
	ContainerSortByIP          ContainerSortProperty = "ip"
	ContainerSortByLastPing    ContainerSortProperty = "last_ping"
	ContainerSortByLastSuccess ContainerSortProperty = "last_success"
)

type ContainerOrder string

const (
	ContainerSortAsc  ContainerOrder = "asc"
	ContainerSortDesc ContainerOrder = "desc"
)

type ContainerInfo struct {
	IP          netip.Addr `json:"ip"`
	LastPing    *time.Time `json:"last_ping,omitempty"`
	LastSuccess *time.Time `json:"last_success,omitempty"`
}
