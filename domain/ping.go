package domain

import (
	"net"
	"time"
)

type Ping struct {
	ID          int
	ContainerIP net.IPAddr `json:"container_ip"`
	Timestamp   *time.Time `json:"timestamp"`
	Success     bool       `json:"success"`
}
