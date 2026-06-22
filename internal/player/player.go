package player

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"sync"
)

type Player struct {
	cmd     *exec.Cmd
	control mpvControl
	done    chan struct{}
	paused  bool
	mu      sync.Mutex
}

func New() *Player {
	return &Player{}
}

func (p *Player) Play(url string) error {
	p.Stop()
	bin, err := exec.LookPath("mpv")
	if err != nil {
		return errors.New("mpv was not found; install mpv to play streams")
	}
	control := newMPVControl()
	args := []string{"--no-video", "--input-terminal=no", "--really-quiet"}
	args = append(args, control.args()...)
	args = append(args, url)
	cmd := exec.Command(bin, args...)
	if err := cmd.Start(); err != nil {
		control.close()
		return err
	}
	done := make(chan struct{})
	p.mu.Lock()
	p.cmd = cmd
	p.control = control
	p.done = done
	p.paused = false
	p.mu.Unlock()
	go func() {
		_ = cmd.Wait()
		control.close()
		p.mu.Lock()
		if p.cmd == cmd {
			p.cmd = nil
			p.control = nil
			p.done = nil
			p.paused = false
		}
		p.mu.Unlock()
		close(done)
	}()
	return nil
}

func (p *Player) TogglePause() (bool, error) {
	p.mu.Lock()
	cmd := p.cmd
	control := p.control
	paused := p.paused
	p.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return false, errors.New("nothing is playing")
	}
	if control == nil {
		return paused, errors.New("mpv control pipe is not available")
	}
	if err := control.command(`{"command":["cycle","pause"]}` + "\n"); err != nil {
		return paused, err
	}
	p.mu.Lock()
	p.paused = !p.paused
	paused = p.paused
	p.mu.Unlock()
	return paused, nil
}

func (p *Player) Title() (string, error) {
	control, ok := p.snapshotControl()
	if !ok {
		return "", errors.New("nothing is playing")
	}
	for _, property := range []string{"metadata/icy-title", "metadata/title", "media-title"} {
		title, err := readProperty(control, property)
		if err == nil && title != "" {
			return title, nil
		}
	}
	return "", nil
}

func (p *Player) Stop() {
	p.mu.Lock()
	cmd := p.cmd
	control := p.control
	done := p.done
	p.cmd = nil
	p.control = nil
	p.done = nil
	p.paused = false
	p.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	if control != nil {
		_ = control.command(`{"command":["quit"]}` + "\n")
	}
	_ = cmd.Process.Kill()
	if done != nil {
		<-done
	}
}

func (p *Player) snapshotControl() (propertyReader, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd == nil || p.cmd.Process == nil || p.control == nil {
		return nil, false
	}
	return p.control, true
}

type propertyReader interface {
	request(string) (string, error)
}

func readProperty(r propertyReader, name string) (string, error) {
	request, err := json.Marshal(map[string]any{"command": []string{"get_property", name}})
	if err != nil {
		return "", err
	}
	raw, err := r.request(string(request) + "\n")
	if err != nil {
		return "", err
	}
	var response struct {
		Data  any    `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		return "", err
	}
	if response.Error != "" && response.Error != "success" {
		return "", errors.New(response.Error)
	}
	title, ok := response.Data.(string)
	if !ok {
		return "", nil
	}
	return strings.TrimSpace(title), nil
}
