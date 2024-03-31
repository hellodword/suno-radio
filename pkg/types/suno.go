package types

import (
	"net/http"
	"time"
)

type DownloadStatus uint8

type PlaylistClip struct {
	ClipMP3Info *ClipMP3Info `json:"clip_mp3_info,omitempty"`
	Clip        struct {
		ID string `json:"id,omitempty"`
		// VideoURL          string `json:"video_url,omitempty"`
		AudioURL string `json:"audio_url,omitempty"`
		// ImageURL          string `json:"image_url,omitempty"`
		// ImageLargeURL     string `json:"image_large_url,omitempty"`
		// MajorModelVersion string `json:"major_model_version,omitempty"`
		// ModelName string `json:"model_name,omitempty"`
		// Metadata struct {
		// 	// Tags   string `json:"tags,omitempty"`
		// 	// Prompt string `json:"prompt,omitempty"`
		// 	// GptDescriptionPrompt any    `json:"gpt_description_prompt,omitempty"`
		// 	// AudioPromptID any `json:"audio_prompt_id,omitempty"`
		// 	// History       any `json:"history,omitempty"`
		// 	ConcatHistory []struct {
		// 		ID         string   `json:"id,omitempty"`
		// 		ContinueAt *float64 `json:"continue_at,omitempty"`
		// 	} `json:"concat_history,omitempty"`
		// 	Type     string  `json:"type,omitempty"`
		// 	Duration float64 `json:"duration,omitempty"`
		// 	// RefundCredits any     `json:"refund_credits,omitempty"`
		// 	// Stream       any `json:"stream,omitempty"`
		// 	// ErrorType    any `json:"error_type,omitempty"`
		// 	// ErrorMessage any `json:"error_message,omitempty"`
		// } `json:"metadata,omitempty"`
		// IsLiked   bool   `json:"is_liked,omitempty"`
		// UserID    string `json:"user_id,omitempty"`
		// IsTrashed bool   `json:"is_trashed,omitempty"`
		// Reaction    any       `json:"reaction,omitempty"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		Status    string    `json:"status,omitempty"`
		Title     string    `json:"title,omitempty"`
		// PlayCount   int64     `json:"play_count,omitempty"`
		UpvoteCount int64 `json:"upvote_count,omitempty"`
		// IsPublic    bool  `json:"is_public,omitempty"`
	} `json:"clip,omitempty"`
	RelativeIndex float64   `json:"relative_index,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
}

type PlaylistClips []*PlaylistClip

func (s PlaylistClips) Len() int {
	return len(s)
}

func (s PlaylistClips) Less(i, j int) bool {
	return s[i].RelativeIndex < s[j].RelativeIndex
}

func (s PlaylistClips) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type Playlist struct {
	ID            string        `json:"id"`
	PlaylistClips PlaylistClips `json:"playlist_clips,omitempty"`
	// ImageURL        string `json:"image_url,omitempty"`
	// NumTotalResults int    `json:"num_total_results,omitempty"`
	// CurrentPage     int    `json:"current_page,omitempty"`
	// IsOwned         bool   `json:"is_owned,omitempty"`
	// IsTrashed       bool   `json:"is_trashed,omitempty"`
	// Name            string `json:"name,omitempty"`
	// Description     string `json:"description,omitempty"`
	// IsPublic        bool   `json:"is_public,omitempty"`
	// IsDiscoverPlaylist any    `json:"is_discover_playlist,omitempty"`
}

type ClipMP3Info struct {
	DataSize   uint64 `json:"data_size,omitempty"`
	DataOffset uint64 `json:"data_offset,omitempty"`
}

type StringSlice []string

func (StringSlice) Render(http.ResponseWriter, *http.Request) error {
	return nil
}
