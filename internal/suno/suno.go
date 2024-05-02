package suno

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/hellodword/suno-radio/internal/ogg"
)

const (
	DefaultSampleRate = 48000
	DefaultChannels   = 2
	DefaultOggSerial  = 1

	ProjectName = "suno-radio"
	ProjectURL  = "https://github.com/hellodword/suno-radio"
)

var PlayListPreset = map[string]string{
	"trending":    "1190bf92-10dc-4ce5-968a-7a377f37f984",
	"new":         "cc14084a-2622-4c4b-8258-1f6b4b4f54b3",
	"weekly":      "08a079b2-a63b-4f9c-9f29-de3c1864ddef",
	"monthly":     "845539aa-2a39-4cf5-b4ae-16d3fe159a77",
	"top":         "6943c7ee-cbc5-4f72-bc4e-f3371a8be9d5",
	"showcase":    "636ed6cb-da70-4123-9ea1-fab61d0165cb",
	"animalparty": "1ac7823f-8faf-474f-b14c-e4f7c7bb373f",
	"lofi":        "6713d315-3541-460d-8788-162cce241336",
}

func verifySunoOgg(p string) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	d := ogg.NewDecoder(f)

	idh, err := d.ParseIDHeader()
	if err != nil {
		return err
	}

	if idh.InputSampleRate != DefaultSampleRate {
		err := fmt.Errorf("sample rate %d", idh.InputSampleRate)
		return err
	}

	if idh.OutputChannelCount != DefaultChannels {
		err := fmt.Errorf("channels %d", idh.OutputChannelCount)
		return err
	}

	// for {
	// 	p, err := d.Decode()
	// 	if err != nil {
	// 		if errors.Is(err, io.EOF) {
	// 			return nil
	// 		}
	// 		return err
	// 	}

	// 	if p.Type != 0 {
	// 		err = fmt.Errorf("invalid suno ogg page type %d", p.Type)
	// 		return err
	// 	}
	// }

	return nil
}
