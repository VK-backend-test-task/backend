package domain

import (
	"fmt"
	"net/netip"
	"time"
)

type ContainerSortProperty string

const (
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

func (c ContainerInfo) String() string {
	lastPing := "NULL"
	if c.LastPing != nil {
		lastPing = c.LastPing.Format(time.RFC3339)
	}
	lastSuccess := "NULL"
	if c.LastSuccess != nil {
		lastSuccess = c.LastSuccess.Format(time.RFC3339)
	}
	return fmt.Sprintf("ContainerInfo { ip: %s, last_ping: %s, last_success: %s }", c.IP.String(), lastPing, lastSuccess)
}
