package mpris

import (
	"github.com/bprendie/subweazl/internal/remote"
)

// Service exposes the live Subweazl player to desktop media clients.
type Service interface {
	Publish(remote.Snapshot)
	Close() error
}
