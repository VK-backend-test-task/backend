package domain

import (
	"fmt"
	"net/netip"
	"time"
)

type Ping struct {
	ID          int
	ContainerIP netip.Addr `json:"container_ip"`
	Timestamp   time.Time  `json:"timestamp"`
	Success     bool       `json:"success"`
}

func (p Ping) String() string {
	success := "false"
	if p.Success {
		success = "true"
	}
	return fmt.Sprintf("Ping { ip: %s, ts: %s, success: %s }", p.ContainerIP.String(), p.Timestamp.Format(time.RFC3339), success)
}
