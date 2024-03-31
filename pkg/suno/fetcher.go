package suno

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

var (
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"
)

func GetPlaylist(ctx context.Context, id string, page uint) (*Playlist, error) {
	if page == 0 {
		page = 1
	}
	u := fmt.Sprintf("https://studio-api.suno.ai/api/playlist/%s/?page=%d", id, page)

	client := &http.Client{
		Transport: http.DefaultTransport,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", DefaultUserAgent)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var p Playlist
	err = json.NewDecoder(res.Body).Decode(&p)
	if err != nil {
		return nil, err
	}

	if p.ID != id {
		err = fmt.Errorf("%s != %s", p.ID, id)
		return nil, err
	}

	return &p, nil
}

func DownloadMP3(ctx context.Context, u, path string) error {

	c := &http.Client{
		Transport: http.DefaultTransport,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", DefaultUserAgent)

	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if !(http.StatusOK <= res.StatusCode && res.StatusCode < http.StatusMultipleChoices) {
		err = fmt.Errorf("status code %d", res.StatusCode)
		return err
	}

	contentType := res.Header.Get("content-type")
	if contentType != "audio/mp3" {
		err = fmt.Errorf("content-type %s", contentType)
		return err
	}

	contentLengthValue := res.Header.Get("content-length")
	contentLength, err := strconv.ParseInt(contentLengthValue, 10, 0)
	if err != nil {
		return err
	}

	if contentLength <= 0 {
		err = fmt.Errorf("content-length %d", contentLength)
		return err
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	written, err := io.Copy(f, res.Body)
	if err != nil {
		return err
	}

	if written != contentLength {
		err = fmt.Errorf("content-length %d written %d", contentLength, written)
		return err
	}

	return nil
}
