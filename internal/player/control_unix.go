//go:build !windows

package player

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type socketControl struct {
	path   string
	tmpDir string
}

func newMPVControl() mpvControl {
	if tmpDir, err := os.MkdirTemp("", "subweazl-*"); err == nil {
		return &socketControl{path: filepath.Join(tmpDir, "mpv.sock"), tmpDir: tmpDir}
	}
	name := "subweazl-" + strconv.Itoa(os.Getpid()) + "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".sock"
	return &socketControl{path: filepath.Join(os.TempDir(), name)}
}

func (c *socketControl) args() []string {
	return []string{"--input-ipc-server=" + c.path}
}

func (c *socketControl) command(command string) error {
	_, err := c.request(command)
	return err
}

func (c *socketControl) request(command string) (string, error) {
	conn, err := net.DialTimeout("unix", c.path, time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		return "", err
	}
	if _, err := conn.Write([]byte(command)); err != nil {
		return "", err
	}
	response, err := bufio.NewReader(conn).ReadString('\n')
	return response, err
}

func (c *socketControl) close() {
	_ = os.Remove(c.path)
	if c.tmpDir != "" {
		_ = os.RemoveAll(c.tmpDir)
	}
}
