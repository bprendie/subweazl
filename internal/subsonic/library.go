package subsonic

import (
	"context"
	"net/url"
	"strconv"
)

func (c Client) Albums(ctx context.Context, offset, size int) ([]Album, error) {
	if size <= 0 {
		size = 500
	}
	var out struct {
		Response struct {
			response
			AlbumList2 struct {
				Album []Album `json:"album"`
			} `json:"albumList2"`
		} `json:"subsonic-response"`
	}
	v := url.Values{
		"type":   {"alphabeticalByName"},
		"size":   {strconv.Itoa(size)},
		"offset": {strconv.Itoa(offset)},
	}
	if err := c.get(ctx, "getAlbumList2", v, &out); err != nil {
		return nil, err
	}
	return out.Response.AlbumList2.Album, nil
}

func (c Client) Starred(ctx context.Context) ([]Track, error) {
	var out struct {
		Response struct {
			response
			Starred struct {
				Song []Track `json:"song"`
			} `json:"starred2"`
		} `json:"subsonic-response"`
	}
	if err := c.get(ctx, "getStarred2", nil, &out); err != nil {
		return nil, err
	}
	return out.Response.Starred.Song, nil
}
