package subsonic

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	base     string
	user     string
	password string
	http     *http.Client
}

type Track struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	AlbumID  string `json:"albumId"`
	CoverID  string `json:"coverArt"`
	Duration int    `json:"duration"`
	Genre    string `json:"genre"`
	Year     int    `json:"year"`
}

type Album struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Artist  string `json:"artist"`
	CoverID string `json:"coverArt"`
	Year    int    `json:"year"`
}

type Playlist struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"songCount"`
}

func New(base, user, password string) Client {
	return Client{base: base, user: user, password: password, http: &http.Client{Timeout: 20 * time.Second}}
}

func (c Client) Ping(ctx context.Context) error {
	var out struct {
		Response response `json:"subsonic-response"`
	}
	return c.get(ctx, "ping", nil, &out)
}

func (c Client) Newest(ctx context.Context) ([]Album, error) {
	return c.albumList(ctx, "newest", 40)
}

func (c Client) RandomAlbums(ctx context.Context) ([]Album, error) {
	return c.albumList(ctx, "random", 40)
}

func (c Client) albumList(ctx context.Context, listType string, size int) ([]Album, error) {
	var out struct {
		Response struct {
			response
			AlbumList2 struct {
				Album []Album `json:"album"`
			} `json:"albumList2"`
		} `json:"subsonic-response"`
	}
	v := url.Values{"type": {listType}, "size": {strconv.Itoa(size)}}
	if err := c.get(ctx, "getAlbumList2", v, &out); err != nil {
		return nil, err
	}
	return out.Response.AlbumList2.Album, nil
}

func (c Client) Album(ctx context.Context, id string) ([]Track, error) {
	var out struct {
		Response struct {
			response
			Album struct {
				Song []Track `json:"song"`
			} `json:"album"`
		} `json:"subsonic-response"`
	}
	if err := c.get(ctx, "getAlbum", url.Values{"id": {id}}, &out); err != nil {
		return nil, err
	}
	return out.Response.Album.Song, nil
}

func (c Client) Playlists(ctx context.Context) ([]Playlist, error) {
	var out struct {
		Response struct {
			response
			Playlists struct {
				Playlist []Playlist `json:"playlist"`
			} `json:"playlists"`
		} `json:"subsonic-response"`
	}
	if err := c.get(ctx, "getPlaylists", nil, &out); err != nil {
		return nil, err
	}
	return out.Response.Playlists.Playlist, nil
}

func (c Client) Playlist(ctx context.Context, id string) ([]Track, error) {
	var out struct {
		Response struct {
			response
			Playlist struct {
				Entry []Track `json:"entry"`
			} `json:"playlist"`
		} `json:"subsonic-response"`
	}
	if err := c.get(ctx, "getPlaylist", url.Values{"id": {id}}, &out); err != nil {
		return nil, err
	}
	return out.Response.Playlist.Entry, nil
}

func (c Client) Search(ctx context.Context, query string) ([]Track, error) {
	var out struct {
		Response struct {
			response
			Search struct {
				Song []Track `json:"song"`
			} `json:"searchResult3"`
		} `json:"subsonic-response"`
	}
	v := url.Values{"query": {query}, "songCount": {"80"}, "albumCount": {"0"}, "artistCount": {"0"}}
	if err := c.get(ctx, "search3", v, &out); err != nil {
		return nil, err
	}
	return out.Response.Search.Song, nil
}

func (c Client) Similar(ctx context.Context, seed Track, limit int) ([]Track, error) {
	seen := map[string]bool{seed.ID: true}
	tracks := []Track{seed}
	if seed.ID != "" {
		similar, err := c.SimilarSongs(ctx, seed.ID, limit)
		if err == nil {
			tracks = appendUnique(tracks, similar, seen, limit)
			if len(tracks) >= limit {
				return tracks, nil
			}
		}
	}
	for _, query := range []string{seed.Artist, seed.Album} {
		if query == "" {
			continue
		}
		found, err := c.Search(ctx, query)
		if err != nil {
			return nil, err
		}
		tracks = appendUnique(tracks, found, seen, limit)
		if len(tracks) >= limit {
			return tracks, nil
		}
	}
	if seed.Genre != "" {
		genreTracks, err := c.RandomSongsByGenre(ctx, seed.Genre, limit)
		if err == nil {
			tracks = appendUnique(tracks, genreTracks, seen, limit)
			if len(tracks) >= limit {
				return tracks, nil
			}
		}
	}
	if seed.Year > 0 {
		yearTracks, err := c.RandomSongsByYear(ctx, seed.Year, limit)
		if err == nil {
			tracks = appendUnique(tracks, yearTracks, seen, limit)
			if len(tracks) >= limit {
				return tracks, nil
			}
		}
	}
	random, err := c.RandomSongs(ctx, limit)
	if err != nil {
		return tracks, nil
	}
	return appendUnique(tracks, random, seen, limit), nil
}

func (c Client) SimilarSongs(ctx context.Context, id string, limit int) ([]Track, error) {
	var out struct {
		Response struct {
			response
			SimilarSongs struct {
				Song []Track `json:"song"`
			} `json:"similarSongs2"`
		} `json:"subsonic-response"`
	}
	v := url.Values{"id": {id}, "count": {strconv.Itoa(limit)}}
	if err := c.get(ctx, "getSimilarSongs2", v, &out); err != nil {
		return nil, err
	}
	return out.Response.SimilarSongs.Song, nil
}

func (c Client) RandomSongs(ctx context.Context, limit int) ([]Track, error) {
	return c.randomSongs(ctx, url.Values{"size": {strconv.Itoa(limit)}})
}

func (c Client) RandomSongsByGenre(ctx context.Context, genre string, limit int) ([]Track, error) {
	return c.randomSongs(ctx, url.Values{"size": {strconv.Itoa(limit)}, "genre": {genre}})
}

func (c Client) RandomSongsByYear(ctx context.Context, year, limit int) ([]Track, error) {
	from := strconv.Itoa(year - 2)
	to := strconv.Itoa(year + 2)
	return c.randomSongs(ctx, url.Values{"size": {strconv.Itoa(limit)}, "fromYear": {from}, "toYear": {to}})
}

func (c Client) randomSongs(ctx context.Context, v url.Values) ([]Track, error) {
	var out struct {
		Response struct {
			response
			RandomSongs struct {
				Song []Track `json:"song"`
			} `json:"randomSongs"`
		} `json:"subsonic-response"`
	}
	if err := c.get(ctx, "getRandomSongs", v, &out); err != nil {
		return nil, err
	}
	return out.Response.RandomSongs.Song, nil
}

func (c Client) CreatePlaylist(ctx context.Context, name string, tracks []Track) (Playlist, error) {
	var out struct {
		Response struct {
			response
			Playlist Playlist `json:"playlist"`
		} `json:"subsonic-response"`
	}
	v := url.Values{"name": {name}}
	for _, track := range tracks {
		if track.ID != "" {
			v.Add("songId", track.ID)
		}
	}
	if err := c.get(ctx, "createPlaylist", v, &out); err != nil {
		return Playlist{}, err
	}
	return out.Response.Playlist, nil
}

func (c Client) RenamePlaylist(ctx context.Context, id, name string) error {
	var out struct {
		Response response `json:"subsonic-response"`
	}
	return c.get(ctx, "updatePlaylist", url.Values{"playlistId": {id}, "name": {name}}, &out)
}

func (c Client) StreamURL(id string) string {
	return c.endpoint("stream", url.Values{"id": {id}}).String()
}

func (c Client) CoverURL(id string) string {
	if id == "" {
		return ""
	}
	return c.endpoint("getCoverArt", url.Values{"id": {id}, "size": {"600"}}).String()
}

func (c Client) CoverArt(ctx context.Context, id string, size int) ([]byte, error) {
	if id == "" {
		return nil, errors.New("missing cover art id")
	}
	if size <= 0 {
		size = 600
	}
	u := c.endpoint("getCoverArt", url.Values{"id": {id}, "size": {strconv.Itoa(size)}})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("subsonic http %d", res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 8<<20))
}

func (c Client) get(ctx context.Context, method string, values url.Values, target any) error {
	u := c.endpoint(method, values)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("subsonic http %d", res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		return err
	}
	if r, ok := findResponse(target); ok && r.Status == "failed" {
		return errors.New(r.Error.Message)
	}
	return nil
}

func (c Client) endpoint(method string, values url.Values) *url.URL {
	u, _ := url.Parse(c.base + "/rest/" + method + ".view")
	q := u.Query()
	for key, vals := range values {
		for _, value := range vals {
			q.Add(key, value)
		}
	}
	salt := strconv.FormatInt(time.Now().UnixNano(), 36)
	sum := md5.Sum([]byte(c.password + salt))
	q.Set("u", c.user)
	q.Set("s", salt)
	q.Set("t", hex.EncodeToString(sum[:]))
	q.Set("v", "1.16.1")
	q.Set("c", "subweazl")
	q.Set("f", "json")
	u.RawQuery = q.Encode()
	return u
}

type response struct {
	Status string `json:"status"`
	Error  struct {
		Message string `json:"message"`
	} `json:"error"`
}

func findResponse(target any) (response, bool) {
	b, _ := json.Marshal(target)
	var out struct {
		Response response `json:"subsonic-response"`
	}
	if json.Unmarshal(b, &out) != nil {
		return response{}, false
	}
	return out.Response, true
}

func appendUnique(dst, src []Track, seen map[string]bool, limit int) []Track {
	for _, track := range src {
		if track.ID == "" || seen[track.ID] {
			continue
		}
		dst = append(dst, track)
		seen[track.ID] = true
		if len(dst) >= limit {
			return dst
		}
	}
	return dst
}
