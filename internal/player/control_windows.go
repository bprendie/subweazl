//go:build windows

package player

import (
	"bufio"
	"os"
	"strconv"
	"time"
)

type namedPipeControl struct {
	path string
}

func newMPVControl() mpvControl {
	name := "subweazl-" + strconv.Itoa(os.Getpid()) + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	return &namedPipeControl{path: `\\.\pipe\` + name}
}

func (c *namedPipeControl) args() []string {
	return []string{"--input-ipc-server=" + c.path}
}

func (c *namedPipeControl) command(command string) error {
	_, err := c.request(command)
	return err
}

func (c *namedPipeControl) request(command string) (string, error) {
	file, err := os.OpenFile(c.path, os.O_RDWR, 0)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.WriteString(command); err != nil {
		return "", err
	}
	response, err := bufio.NewReader(file).ReadString('\n')
	return response, err
}

func (c *namedPipeControl) close() {}
