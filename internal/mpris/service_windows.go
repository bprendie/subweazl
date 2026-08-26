//go:build windows

package mpris

import (
	"errors"

	"github.com/bprendie/subweazl/internal/remote"
)

type unsupportedService struct{}

func Start(func(remote.Command)) (Service, error) {
	return nil, errors.New("MPRIS is not available on Windows")
}

func (unsupportedService) Publish(remote.Snapshot) {}
func (unsupportedService) Close() error            { return nil }
