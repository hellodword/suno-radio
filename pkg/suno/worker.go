package suno

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/hellodword/suno-radio/3rd/go-mp3"
)

type Worker struct {
	id           string
	playlist     *Playlist
	playlistLock sync.Mutex

	dir      string
	interval time.Duration

	wg     sync.WaitGroup
	logger *slog.Logger
}

func NewWorker(ctx context.Context, logger *slog.Logger, id string, interval time.Duration, dir string) (*Worker, error) {
	var err error

	w := &Worker{id: id, interval: interval, dir: dir, logger: logger}

	w.logger.InfoContext(ctx, "fetching playlist")
	// TODO pagination
	w.playlist, err = GetPlaylist(ctx, id, 1)
	if err != nil {
		w.logger.ErrorContext(ctx, "fetch playlist", "err", err)
		return nil, err
	}
	w.logger.InfoContext(ctx, "fetched playlist")

	return w, nil
}

func (w *Worker) ID() string { return w.id }

func (w *Worker) Start(ctx context.Context) {

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		fetch := func() error {
			w.logger.InfoContext(ctx, "fetching playlist")
			playlist, err := GetPlaylist(ctx, w.id, 1)
			if err != nil {
				w.logger.ErrorContext(ctx, "fetch playlist", "err", err)
				return err
			}
			w.logger.InfoContext(ctx, "fetched playlist")

			// remove outdated clips
			for i := 0; i < len(w.playlist.PlaylistClips); i++ {
				found := false
				for _, newClip := range playlist.PlaylistClips {
					if newClip.Clip.ID == w.playlist.PlaylistClips[i].Clip.ID {
						found = true
						break
					}
				}
				if !found {
					w.playlistLock.Lock()
					w.playlist.PlaylistClips = append(w.playlist.PlaylistClips[:i], w.playlist.PlaylistClips[i+1:]...)
					w.playlistLock.Unlock()
					i--
				}
			}

			for _, newClip := range playlist.PlaylistClips {
				found := false
				for i := range w.playlist.PlaylistClips {
					if newClip.Clip.ID == w.playlist.PlaylistClips[i].Clip.ID {
						found = true
						w.playlist.PlaylistClips[i].RelativeIndex = newClip.RelativeIndex
						break
					}
				}

				if !found {
					w.playlistLock.Lock()
					w.playlist.PlaylistClips = append(w.playlist.PlaylistClips, newClip)
					w.playlistLock.Unlock()
				}
			}

			sort.Sort(playlist.PlaylistClips)

			return nil
		}

		parse := func(p string) (uint64, uint64, error) {
			f, err := os.Open(p)
			if err != nil {
				return 0, 0, err
			}

			d, err := mp3.NewDecoder(f)
			if err != nil {
				return 0, 0, err
			}

			if d.SampleRate() != DefaultSampleRate {
				err := fmt.Errorf("sample rate %d", d.SampleRate())
				return 0, 0, err
			}

			dataOffset, dataSize := d.DataOffset()
			return dataOffset, dataSize, nil
		}

		download := func() {

			var clips = w.playlist.PlaylistClips
			// if len(w.playlist.PlaylistClips) > 20 {
			// }

			var filenames []string
			for _, clip := range clips {
				filenames = append(filenames, clip.Clip.ID+".mp3")
			}

			filepath.WalkDir(w.dir, func(p string, d fs.DirEntry, err error) error {
				if !d.IsDir() && filepath.Ext(p) == ".mp3" {
					base := filepath.Base(p)
					found := false
					for _, filename := range filenames {
						if filename == base {
							found = true
							break
						}
					}
					if !found {
						w.logger.InfoContext(ctx, "delete unused file", "base", base)
						if err := os.Remove(p); err != nil {
							w.logger.ErrorContext(ctx, "delete unused file", "base", base, "err", err)
						}
					}
				}
				return nil
			})

			for _, clip := range clips {

				select {
				case <-ctx.Done():
					w.logger.InfoContext(ctx, "exit")
					return
				default:
				}

				p := path.Join(w.dir, fmt.Sprintf("%s.mp3", clip.Clip.ID))

				shouldDownload := false
				stat, err := os.Stat(p)
				if errors.Is(err, os.ErrNotExist) {
					shouldDownload = true
				} else if stat != nil && clip.ClipMP3Info != nil {
					minSize := clip.ClipMP3Info.DataOffset + clip.ClipMP3Info.DataSize
					if stat.Size() < int64(minSize) {
						w.logger.ErrorContext(ctx, "mp3 size not match", "p", p)
						shouldDownload = true
					}
				}

				if shouldDownload {
					clip.ClipMP3Info = nil
					w.logger.InfoContext(ctx, "downloading mp3", "p", p)
					err := DownloadMP3(ctx, clip.Clip.AudioURL, p)
					if err != nil {
						w.logger.ErrorContext(ctx, "download mp3", "p", p, "err", err)
						continue
					}
					w.logger.InfoContext(ctx, "downloaded mp3", "p", p)
				}

				if clip.ClipMP3Info == nil {
					dataOffset, dataSize, err := parse(p)
					if err != nil {
						w.logger.ErrorContext(ctx, "parse mp3", "p", p, "err", err)
						continue
					}

					clip.ClipMP3Info = &ClipMP3Info{
						DataOffset: dataOffset,
						DataSize:   dataSize,
					}
				}
			}

		}

		download()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := fetch()
				if err != nil {
					continue
				}
				download()
			}
		}

	}()

}

func (w *Worker) Close() error {
	w.wg.Wait()
	return nil
}

func (w *Worker) Stream(ctx context.Context, writer io.Writer) error {
	// TODO use buffer instead of reading files?

	err := writeFull(writer, MP3Header)
	if err != nil {
		return err
	}

	// TODO prefer the latest clip instead of a random clip
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	wait := func(t time.Duration) {
		timer := time.NewTimer(t)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		var clip *PlaylistClip
		w.playlistLock.Lock()
		l := len(w.playlist.PlaylistClips)
		if l > 0 {
			clip = w.playlist.PlaylistClips[r.Intn(l-1)]
		}
		w.playlistLock.Unlock()

		if clip == nil || clip.ClipMP3Info == nil {
			wait(time.Millisecond * 100)
			continue
		}

		err = w.streamClip(ctx, writer, clip)
		if err != nil {
			return err
		}
	}
}

func (w *Worker) streamClip(ctx context.Context, writer io.Writer, clip *PlaylistClip) error {
	p := path.Join(w.dir, fmt.Sprintf("%s.mp3", clip.Clip.ID))
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}
	minSize := clip.ClipMP3Info.DataOffset + clip.ClipMP3Info.DataSize
	if stat.Size() < int64(minSize) {
		err = fmt.Errorf("file size %s", clip.Clip.ID)
		// delete file with wrong size
		defer os.Remove(p)
		return err
	}

	pos, err := f.Seek(int64(clip.ClipMP3Info.DataOffset), io.SeekStart)
	if err != nil {
		return err
	}

	if pos != int64(clip.ClipMP3Info.DataOffset) {
		err = fmt.Errorf("seek %s", clip.Clip.ID)
		return err
	}

	_, err = io.CopyN(writer, f, int64(clip.ClipMP3Info.DataSize))
	if err != nil {
		return err
	}

	return w.paddingSilence(ctx, writer)
}

func (w *Worker) paddingSilence(ctx context.Context, writer io.Writer) error {
	for range 30 { // 3s silence
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		err := writeFull(writer, MP3Silence100ms[len(MP3Header):])
		if err != nil {
			return err
		}
	}
	return nil
}

func writeFull(w io.Writer, b []byte) error {
	l := len(b)
	written, err := w.Write(b)
	if err != nil {
		return err
	}

	if written != l {
		err = fmt.Errorf("write full failed %d/%d", written, l)
		return err
	}

	return nil
}
