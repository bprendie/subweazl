//go:build !windows

package remote

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

type Server struct {
	listener net.Listener
	path     string
}

func Listen(handler func(Command)) (*Server, error) {
	path, err := SocketPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if conn, err := net.DialTimeout("unix", path, 100*time.Millisecond); err == nil {
		conn.Close()
		return nil, fmt.Errorf("Subweazl is already running")
	}
	_ = os.Remove(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(path, 0o600)
	server := &Server{listener: listener, path: path}
	go server.serve(handler)
	return server, nil
}

func (s *Server) serve(handler func(Command)) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go handleConnection(conn, handler)
	}
}

func handleConnection(conn net.Conn, handler func(Command)) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	var request struct {
		Command Command `json:"command"`
	}
	err := json.NewDecoder(bufio.NewReader(conn)).Decode(&request)
	if err == nil && validCommand(request.Command) {
		handler(request.Command)
	} else if err == nil {
		err = fmt.Errorf("unknown remote command %q", request.Command)
	}
	_ = json.NewEncoder(conn).Encode(map[string]any{"ok": err == nil, "error": errorText(err)})
}

func Send(command Command) error {
	path, err := SocketPath()
	if err != nil {
		return err
	}
	conn, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		return fmt.Errorf("Subweazl is not running")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if err := json.NewEncoder(conn).Encode(map[string]Command{"command": command}); err != nil {
		return err
	}
	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("%s", response.Error)
	}
	return nil
}

func (s *Server) Close() error {
	err := s.listener.Close()
	_ = os.Remove(s.path)
	return err
}

func validCommand(command Command) bool {
	return command == Toggle || command == Next || command == Previous || command == Stop || command == CycleMode || command == Quit
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
