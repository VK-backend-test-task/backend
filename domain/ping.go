package domain

import (
	"net/netip"
	"time"
)

type Ping struct {
	ID          int
	ContainerIP netip.Addr `json:"container_ip"`
	Timestamp   time.Time  `json:"timestamp"`
	Success     bool       `json:"success"`
}
