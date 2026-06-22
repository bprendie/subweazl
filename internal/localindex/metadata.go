package localindex

import (
	"encoding/json"
	"strconv"
	"strings"
)

type Metadata struct {
	Path            string  `json:"path"`
	Title           string  `json:"title,omitempty"`
	Artist          string  `json:"artist,omitempty"`
	Album           string  `json:"album,omitempty"`
	AlbumArtist     string  `json:"album_artist,omitempty"`
	Genre           string  `json:"genre,omitempty"`
	Year            int     `json:"year,omitempty"`
	TrackNumber     int     `json:"track_number,omitempty"`
	DiscNumber      int     `json:"disc_number,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	Codec           string  `json:"codec,omitempty"`
	Container       string  `json:"container,omitempty"`
	FileSize        int64   `json:"file_size,omitempty"`
	ModifiedUnix    int64   `json:"modified_unix,omitempty"`
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
}

type ffprobeFormat struct {
	FormatName string            `json:"format_name"`
	Duration   string            `json:"duration"`
	Tags       map[string]string `json:"tags"`
}

func ParseFFProbe(data []byte) (Metadata, error) {
	var out ffprobeOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return Metadata{}, err
	}
	tags := normalizedTags(out.Format.Tags)
	meta := Metadata{
		Title:       first(tags, "title"),
		Artist:      first(tags, "artist"),
		Album:       first(tags, "album"),
		AlbumArtist: first(tags, "album_artist", "albumartist"),
		Genre:       first(tags, "genre"),
		Year:        parseYear(first(tags, "date", "year", "originaldate")),
		TrackNumber: parseLeadingInt(first(tags, "track", "tracknumber")),
		DiscNumber:  parseLeadingInt(first(tags, "disc", "discnumber")),
		Container:   out.Format.FormatName,
	}
	if dur, err := strconv.ParseFloat(out.Format.Duration, 64); err == nil {
		meta.DurationSeconds = dur
	}
	for _, stream := range out.Streams {
		if stream.CodecType == "audio" {
			meta.Codec = stream.CodecName
			break
		}
	}
	return meta, nil
}

func normalizedTags(tags map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range tags {
		out[strings.ToLower(strings.ReplaceAll(key, "-", "_"))] = strings.TrimSpace(value)
	}
	return out
}

func first(tags map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := tags[key]; value != "" {
			return value
		}
	}
	return ""
}

func parseLeadingInt(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	for i, r := range value {
		if r < '0' || r > '9' {
			value = value[:i]
			break
		}
	}
	n, _ := strconv.Atoi(value)
	return n
}

func parseYear(value string) int {
	for i := 0; i+4 <= len(value); i++ {
		if year, err := strconv.Atoi(value[i : i+4]); err == nil && year > 0 {
			return year
		}
	}
	return 0
}
