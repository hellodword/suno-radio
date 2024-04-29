package suno

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math/rand"
	"net/rpc"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/go-audio/wav"
	"github.com/hellodword/suno-radio/internal/common"
)

type Worker struct {
	id           string
	playlist     *Playlist
	playlistLock sync.Mutex

	dir      string
	interval time.Duration

	wg     sync.WaitGroup
	logger *slog.Logger

	converterRpc *rpc.Client
}

func NewWorker(ctx context.Context, logger *slog.Logger, id string, interval time.Duration, dir string, converterRpc *rpc.Client) (*Worker, error) {
	var err error

	w := &Worker{id: id, interval: interval, dir: dir, logger: logger, converterRpc: converterRpc}

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

func (w *Worker) ID() string         { return w.id }
func (w *Worker) Info() PlaylistInfo { return w.playlist.PlaylistInfo }

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

		parseWav := func(p string) (uint64, uint64, error) {
			f, err := os.Open(p)
			if err != nil {
				return 0, 0, err
			}
			defer f.Close()

			d := wav.NewDecoder(f)
			d.ReadInfo()
			if d.SampleRate != DefaultWavSampleRate {
				err := fmt.Errorf("sample rate %d", d.SampleRate)
				return 0, 0, err
			}

			err = d.FwdToPCM()
			if err != nil {
				return 0, 0, err
			}

			pos, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				return 0, 0, err
			}

			return uint64(pos), uint64(d.PCMLen()), nil
		}

		isFilePrepared := func(clip *PlaylistClip) (downloaded, converted bool) {
			pmp3 := path.Join(w.dir, fmt.Sprintf("%s.mp3", clip.Clip.ID))
			pwav := path.Join(w.dir, fmt.Sprintf("%s.wav", clip.Clip.ID))
			stat, err := os.Stat(pmp3)
			downloaded = err == nil && stat != nil && !stat.IsDir()
			if !downloaded {
				return
			}

			stat, err = os.Stat(pwav)

			converted = err == nil && stat != nil && !stat.IsDir()

			return
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

			// downloaded clips
			for _, clip := range clips {

				select {
				case <-ctx.Done():
					w.logger.InfoContext(ctx, "exit")
					return
				default:
				}

				pmp3 := path.Join(w.dir, fmt.Sprintf("%s.mp3", clip.Clip.ID))
				pwav := path.Join(w.dir, fmt.Sprintf("%s.wav", clip.Clip.ID))

				downloaded, converted := isFilePrepared(clip)
				if !downloaded {
					os.Remove(pmp3)
					continue
				}

				if !converted {
					w.logger.InfoContext(ctx, "converting mp3 to wav", "p", pmp3)
					err := common.MP3ToWavConverterConvert(w.converterRpc, common.MP3ToWavConverterArgs{
						Playlist: w.id,
						ClipID:   clip.Clip.ID,
					})
					if err != nil {
						os.Remove(pwav)
						w.logger.ErrorContext(ctx, "convert mp3 to wav", "p", pmp3, "err", err)
						continue
					}
					w.logger.InfoContext(ctx, "converted mp3 to wav", "p", pmp3)
				}

				if clip.ClipWavInfo == nil {
					dataOffset, dataSize, err := parseWav(pwav)
					if err != nil {
						os.Remove(pwav)
						w.logger.ErrorContext(ctx, "parse wav", "p", pwav, "err", err)
						continue
					}

					w.logger.InfoContext(ctx, "added", "id", clip.Clip.ID, "offset", dataOffset, "data size", dataSize)
					clip.ClipWavInfo = &ClipWavInfo{
						DataOffset: dataOffset,
						DataSize:   dataSize,
					}
				}
			}

			for _, clip := range clips {

				select {
				case <-ctx.Done():
					w.logger.InfoContext(ctx, "exit")
					return
				default:
				}

				if clip.ClipWavInfo != nil {
					continue
				}

				pmp3 := path.Join(w.dir, fmt.Sprintf("%s.mp3", clip.Clip.ID))
				pwav := path.Join(w.dir, fmt.Sprintf("%s.wav", clip.Clip.ID))

				downloaded, converted := isFilePrepared(clip)
				if !downloaded {

					w.logger.InfoContext(ctx, "downloading mp3", "p", pmp3)
					err := DownloadMP3(ctx, clip.Clip.AudioURL, pmp3)
					if err != nil {
						w.logger.ErrorContext(ctx, "download mp3", "p", pmp3, "err", err)
						continue
					}
					w.logger.InfoContext(ctx, "downloaded mp3", "p", pmp3)
				}

				if !converted {
					os.Remove(pwav)

					w.logger.InfoContext(ctx, "converting mp3 to wav", "p", pmp3)
					err := common.MP3ToWavConverterConvert(w.converterRpc, common.MP3ToWavConverterArgs{
						Playlist: w.id,
						ClipID:   clip.Clip.ID,
					})
					if err != nil {
						os.Remove(pwav)
						w.logger.ErrorContext(ctx, "convert mp3 to wav", "p", pmp3, "err", err)
						continue
					}
					w.logger.InfoContext(ctx, "converted mp3 to wav", "p", pmp3)
				}

				if clip.ClipWavInfo == nil {
					dataOffset, dataSize, err := parseWav(pwav)
					if err != nil {
						w.logger.ErrorContext(ctx, "parse wav", "p", pwav, "err", err)
						continue
					}

					w.logger.InfoContext(ctx, "added", "id", clip.Clip.ID, "offset", dataOffset, "data size", dataSize)
					clip.ClipWavInfo = &ClipWavInfo{
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

	err := writeFull(writer, infiniteWavHeader)
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

		if clip == nil || clip.ClipWavInfo == nil {
			wait(time.Millisecond * 100)
			continue
		}

		w.logger.InfoContext(ctx, "streaming clip",
			"clip", clip.Clip.ID,
			"offset", clip.ClipWavInfo.DataOffset,
			"size", clip.ClipWavInfo.DataSize)
		err = w.streamClip(ctx, writer, clip)
		if err != nil {
			return err
		}
	}
}

func (w *Worker) streamClip(ctx context.Context, writer io.Writer, clip *PlaylistClip) error {
	pwav := path.Join(w.dir, fmt.Sprintf("%s.wav", clip.Clip.ID))
	f, err := os.Open(pwav)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}
	minSize := clip.ClipWavInfo.DataOffset + clip.ClipWavInfo.DataSize
	if stat.Size() < int64(minSize) {
		err = fmt.Errorf("file size %s", clip.Clip.ID)
		// delete file with wrong size
		defer os.Remove(pwav)
		return err
	}

	pos, err := f.Seek(int64(clip.ClipWavInfo.DataOffset), io.SeekStart)
	if err != nil {
		return err
	}

	if pos != int64(clip.ClipWavInfo.DataOffset) {
		err = fmt.Errorf("seek %s", clip.Clip.ID)
		return err
	}

	_, err = io.CopyN(writer, f, int64(clip.ClipWavInfo.DataSize))
	if err != nil {
		return err
	}

	return w.paddingSilence(ctx, writer, time.Second*3)
}

func (w *Worker) paddingSilence(ctx context.Context, writer io.Writer, duration time.Duration) error {
	count := 1
	if duration > time.Millisecond*100 {
		count = int(duration / (time.Millisecond * 100))
	}
	for range count {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		err := writeFull(writer, PCMSilence100ms)
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
