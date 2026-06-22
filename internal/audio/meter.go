package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strings"
	"sync"
)

type Sample struct {
	Level     float64
	Transient float64
	Bands     []float64
	Live      bool
}

type Meter struct {
	cmd      *exec.Cmd
	done     chan struct{}
	out      chan Sample
	errs     chan error
	stopping bool
	mu       sync.Mutex
}

func StartMeter(url string) (*Meter, error) {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, errors.New("ffmpeg not found")
	}
	args := []string{
		"-nostdin",
		"-hide_banner",
		"-loglevel", "error",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "2",
		"-re",
		"-i", url,
		"-vn",
		"-f", "s16le",
		"-ac", "1",
		"-ar", "44100",
		"pipe:1",
	}
	cmd := exec.Command(bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	m := &Meter{cmd: cmd, done: make(chan struct{}), out: make(chan Sample, 8), errs: make(chan error, 1)}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go m.read(stdout, &stderr)
	return m, nil
}

func (m *Meter) Samples() <-chan Sample {
	return m.out
}

func (m *Meter) Errors() <-chan error {
	return m.errs
}

func (m *Meter) Stop() {
	m.mu.Lock()
	cmd := m.cmd
	done := m.done
	m.stopping = true
	m.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	<-done
}

func (m *Meter) read(r io.Reader, stderr *bytes.Buffer) {
	defer close(m.done)
	defer close(m.out)
	defer close(m.errs)
	defer m.wait(stderr)
	analyzer := NewSpectrumAnalyzer(44100, 24, 20, 18000)
	buf := make([]byte, 4096)
	previous := 0.0
	for {
		n, err := io.ReadFull(r, buf)
		if err != nil {
			return
		}
		level := rms(buf[:n])
		sample := Sample{
			Level:     level,
			Transient: math.Max(0, level-previous),
			Bands:     analyzer.Bands(buf[:n]),
			Live:      true,
		}
		previous = level*0.72 + previous*0.28
		select {
		case m.out <- sample:
		default:
		}
	}
}

func (m *Meter) wait(stderr *bytes.Buffer) {
	m.mu.Lock()
	cmd := m.cmd
	stopping := m.stopping
	m.cmd = nil
	m.mu.Unlock()
	if cmd == nil {
		return
	}
	err := cmd.Wait()
	if err == nil || stopping {
		return
	}
	detail := strings.TrimSpace(stderr.String())
	if detail != "" {
		err = fmt.Errorf("%w: %s", err, detail)
	}
	select {
	case m.errs <- err:
	default:
	}
}

func rms(buf []byte) float64 {
	if len(buf) < 2 {
		return 0
	}
	total := 0.0
	count := 0
	for i := 0; i+1 < len(buf); i += 2 {
		v := float64(int16(binary.LittleEndian.Uint16(buf[i:]))) / 32768
		total += v * v
		count++
	}
	return math.Min(1, math.Sqrt(total/float64(count))*3.2)
}
