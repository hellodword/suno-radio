package suno

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hellodword/suno-radio/internal/mp3toogg"
	"github.com/hellodword/suno-radio/internal/ogg"
	"github.com/teivah/broadcast"
)

type Worker struct {
	id       string
	alias    string
	playlist *Playlist

	dir      string
	interval time.Duration

	wg     sync.WaitGroup
	logger *slog.Logger

	convertedClips sync.Map

	broadcaster *broadcast.Relay[*oggPage]

	granule   int64
	beginTime time.Time

	streamCount int32

	listeningCLipID atomic.Value

	canceled int32
}

func NewWorker(ctx context.Context, logger *slog.Logger, id, alias string, interval time.Duration, dir string) (*Worker, error) {
	var err error

	w := &Worker{id: id, alias: alias, interval: interval, dir: dir, logger: logger,
		broadcaster: broadcast.NewRelay[*oggPage](),
	}

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

func (w *Worker) ID() string    { return w.id }
func (w *Worker) Alias() string { return w.alias }
func (w *Worker) Info() map[string]any {

	m := map[string]any{
		"info":     w.playlist.PlaylistInfo,
		"listener": atomic.LoadInt32(&w.streamCount),
	}

	listener := atomic.LoadInt32(&w.streamCount)
	if listener > 0 {
		m["listener"] = listener
		listeningV := w.listeningCLipID.Load()
		if listeningV != nil {
			if clip, ok := listeningV.(*PlaylistClip); ok {
				var listening = map[string]any{
					"title":        clip.Clip.Title,
					"upvote_count": clip.Clip.UpvoteCount,
					"url":          fmt.Sprintf("https://suno.com/song/%s", clip.Clip.ID),
				}

				if clip.Clip.Title != "" {
					listening["title"] = clip.Clip.Title
				}

				m["listening"] = listening
			}
		}
	}

	return m
}

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

			w.playlist = playlist

			// remove outdated
			w.convertedClips.Range(func(key, _ any) bool {

				for i := range w.playlist.PlaylistClips {
					if key.(string) == w.playlist.PlaylistClips[i].Clip.ID {
						return true
					}
				}

				w.convertedClips.Delete(key.(string))

				pmp3 := path.Join(w.dir, fmt.Sprintf("%s.mp3", key.(string)))
				pogg := path.Join(w.dir, fmt.Sprintf("%s.ogg", key.(string)))

				os.Remove(pmp3)
				os.Remove(pogg)
				os.Remove(pmp3 + ".tmp")

				return true
			})

			return nil
		}

		isFilePrepared := func(clip *PlaylistClip) (downloaded, converted bool) {
			pmp3 := path.Join(w.dir, fmt.Sprintf("%s.mp3", clip.Clip.ID))
			pogg := path.Join(w.dir, fmt.Sprintf("%s.ogg", clip.Clip.ID))

			stat, err := os.Stat(pogg)
			converted = err == nil && stat != nil && !stat.IsDir() && verifySunoOgg(pogg) == nil
			if converted {
				downloaded = true
				return
			}

			stat, err = os.Stat(pmp3)
			downloaded = err == nil && stat != nil && !stat.IsDir()
			if !downloaded {
				return
			}

			return
		}

		download := func() {

			var clipsDownloaded, clipsNotDownloaded []*PlaylistClip

			// downloaded clips
			for _, clip := range w.playlist.PlaylistClips {
				downloaded, _ := isFilePrepared(clip)
				if downloaded {
					clipsDownloaded = append(clipsDownloaded, clip)
				} else {
					clipsNotDownloaded = append(clipsNotDownloaded, clip)
				}
			}
			for _, clip := range append(clipsDownloaded, clipsNotDownloaded...) {
				if atomic.LoadInt32(&w.canceled) != 0 {
					return
				}

				select {
				case <-ctx.Done():
					return
				default:
				}

				if _, ok := w.convertedClips.Load(clip.Clip.ID); ok {
					continue
				}

				pmp3 := path.Join(w.dir, fmt.Sprintf("%s.mp3", clip.Clip.ID))
				pogg := path.Join(w.dir, fmt.Sprintf("%s.ogg", clip.Clip.ID))

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
					w.logger.InfoContext(ctx, "converting mp3 to ogg", "p", pmp3)
					_, err := mp3toogg.MP3ToOggConvert(mp3toogg.MP3ToOggArgs{
						Playlist:   w.id,
						ClipID:     clip.Clip.ID,
						SampleRate: DefaultSampleRate,
						Channels:   DefaultChannels,
					})
					if err != nil {
						os.Remove(pogg)
						w.logger.ErrorContext(ctx, "convert mp3 to ogg", "p", pmp3, "err", err)
						continue
					}
					w.logger.InfoContext(ctx, "converted mp3 to ogg", "p", pmp3)
				}

				w.convertedClips.Store(clip.Clip.ID, clip)

			}

		}

		download()

		for {
			if atomic.LoadInt32(&w.canceled) != 0 {
				return
			}

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

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		for {
			time.Sleep(time.Millisecond * 50)

			if atomic.LoadInt32(&w.canceled) != 0 {
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			if atomic.LoadInt32(&w.streamCount) < 1 {
				continue
			}

			var clipIDs []string
			w.convertedClips.Range(func(key, _ any) bool {
				clipIDs = append(clipIDs, key.(string))
				return true
			})

			if len(clipIDs) == 0 {
				continue
			}

			clipID := clipIDs[rnd.Intn(len(clipIDs))]

			clipV, ok := w.convertedClips.Load(clipID)
			if !ok {
				continue
			}

			clip := clipV.(*PlaylistClip)

			pogg := path.Join(w.dir, fmt.Sprintf("%s.ogg", clip.Clip.ID))

			f, err := os.Open(pogg)
			if err != nil {
				w.logger.ErrorContext(ctx, "open ogg", "p", pogg, "err", err)
				continue
			}

			w.logger.InfoContext(ctx, "streaming ogg", "p", pogg)
			w.listeningCLipID.Store(clip)
			err = w.streamOgg(ctx, f)
			f.Close()
			if err != nil {
				w.logger.ErrorContext(ctx, "stream ogg", "p", pogg, "err", err)
				continue
			}

			// TODO silence

		}

	}()

}

func (w *Worker) Close() error {
	atomic.StoreInt32(&w.canceled, 1)
	w.broadcaster.Close()
	w.wg.Wait()
	return nil
}

type oggPage struct {
	granule int64
	packets [][]byte
}

func (w *Worker) Stream(id string, ctx context.Context, writer io.Writer) error {

	oggwriter := ogg.NewEncoder(DefaultOggSerial, writer)

	listener := w.broadcaster.Listener(1)
	defer listener.Close()
	w.logger.Info("stream created", "stream id", id)
	defer w.logger.Info("stream exited", "stream id", id)

	atomic.AddInt32(&w.streamCount, 1)
	defer atomic.AddInt32(&w.streamCount, -1)

	idh := &ogg.IDHeader{
		Version:            1,
		OutputChannelCount: DefaultChannels,
		PreSkip:            0,
		InputSampleRate:    DefaultSampleRate,
	}

	packets, err := idh.Encode()
	if err != nil {
		return err
	}

	err = oggwriter.EncodeBOS(0, packets)
	if err != nil {
		return err
	}

	cmh := &ogg.CommentHeader{
		VendorString: ProjectName,
		UserCommentList: map[string]string{
			"CONTACT": ProjectURL,
		},
	}

	packets, err = cmh.Encode()
	if err != nil {
		return err
	}

	err = oggwriter.Encode(0, packets)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case page := <-listener.Ch():
			{
				if page == nil {
					// ???
					return context.Canceled
				}

				if atomic.LoadInt32(&w.canceled) != 0 {
					return context.Canceled
				}

				w.logger.Debug("Subscribe got msg", "stream id", id, "granule", page.granule)
				defer w.logger.Debug("Subscribe got msg exited", "stream id", id, "granule", page.granule)

				err := oggwriter.Encode(page.granule, page.packets)
				if err != nil {
					return err
				}
			}
		}
	}

}

func (w *Worker) streamOgg(ctx context.Context, f io.Reader) error {
	d := ogg.NewDecoder(f)

	var lastGranule int64

	for atomic.LoadInt32(&w.streamCount) > 0 {
		if atomic.LoadInt32(&w.canceled) != 0 {
			return context.Canceled
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		p, err := d.Decode()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			w.logger.ErrorContext(ctx, "open decode", "err", err)
			return err
		}

		if p.Type&ogg.BOS == ogg.BOS {
			continue
		}

		if p.Type&ogg.COP == ogg.COP {
			w.logger.WarnContext(ctx, "ogg COP page")
			continue
		}

		var cmh = &ogg.CommentHeader{}
		if cmh.Decode(p.Packets) == nil {
			w.logger.DebugContext(ctx, "ogg CommentHeader", "comment header", cmh)
			continue
		}

		if len(p.Packets) == 0 {
			continue
		}

		if p.Type&ogg.EOS == ogg.EOS {
			w.logger.DebugContext(ctx, "ogg EOS")
			// TODO padding EOS page's PCM to 'full' opus page size
		}

		pcmLen := p.Granule - lastGranule
		if pcmLen == 0 {
			continue
		}

		if w.granule == 0 {
			w.beginTime = time.Now()
		}

		w.granule += pcmLen
		w.logger.DebugContext(ctx, "publishing ogg page", "len", pcmLen, "granule", w.granule)
		w.broadcaster.Broadcast(&oggPage{
			granule: w.granule,
			packets: p.Packets,
		})
		w.logger.DebugContext(ctx, "published ogg page", "len", pcmLen, "granule", w.granule)
		lastGranule = p.Granule

		// make clients' memory happy
		time.Sleep(time.Millisecond * 900 * time.Duration(pcmLen) / 48000)
		ms := time.Duration(w.granule) * 1000 * time.Millisecond / 48000
		expect := w.beginTime.Add(ms)
		sub := time.Until(expect)
		if sub > time.Millisecond*2000 {
			wait := sub - time.Millisecond*2000
			time.Sleep(wait)
		}

	}

	return nil
}
