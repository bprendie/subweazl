package localindex

import (
	"context"
	"os/exec"
)

type FFProbe struct {
	Path string
}

func (p FFProbe) Probe(ctx context.Context, path string) (Metadata, error) {
	name := p.Path
	if name == "" {
		name = "ffprobe"
	}
	out, err := exec.CommandContext(
		ctx,
		name,
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	).Output()
	if err != nil {
		return Metadata{}, err
	}
	meta, err := ParseFFProbe(out)
	if err != nil {
		return Metadata{}, err
	}
	meta.Path = path
	return meta, nil
}
