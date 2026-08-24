//go:build windows

package remote

import "errors"

type Server struct{}

func Listen(func(Command)) (*Server, error) {
	return nil, errors.New("Subweazl remote control is not available on Windows")
}

func Send(Command) error {
	return errors.New("Subweazl remote control is not available on Windows")
}

func (s *Server) Close() error { return nil }
