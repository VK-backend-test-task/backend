package service

import (
	"backend/repository"
)

// right now it is so small, we can skip re-writing the same interface for right now
type PingService interface {
	repository.PingRepository
}
