package platforms

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"

	"main/internal/config"
	state "main/internal/core/models"
)

const PlatformShrutiApi state.PlatformName = "ShrutiApi"

type ShrutiApiPlatform struct {
	name state.PlatformName
}

func init() {
	Register(80, &ShrutiApiPlatform{
		name: PlatformShrutiApi,
	})
}

func (f *ShrutiApiPlatform) Name() state.PlatformName {
	return f.name
}

func (f *ShrutiApiPlatform) CanGetTracks(query string) bool {
	return false
}

func (f *ShrutiApiPlatform) GetTracks(_ string, _ bool) ([]*state.Track, error) {
	return nil, errors.New("shrutiapi is a download-only platform")
}

func (f *ShrutiApiPlatform) CanDownload(source state.PlatformName) bool {
	if config.ShrutiAPIURL == "" || config.ShrutiAPIKey == "" {
		return false
	}
	return source == PlatformYouTube
}

func (f *ShrutiApiPlatform) Download(
	ctx context.Context,
	track *state.Track,
	_ *telegram.NewMessage,
) (string, error) {
	if cached := findFile(track); cached != "" {
		gologging.Debug("ShrutiApi: Download -> Cached File -> " + cached)
		return cached, nil
	}

	return f.download(ctx, track)
}

func (*ShrutiApiPlatform) CanSearch() bool { return false }

func (*ShrutiApiPlatform) Search(string, bool) ([]*state.Track, error) {
	return nil, nil
}

func (f *ShrutiApiPlatform) download(ctx context.Context, track *state.Track) (string, error) {
	videoID := track.ID
	if videoID == "" {
		videoID = track.URL
	}

	mediaType := "audio"
	ext := ".mp3"
	if track.Video {
		mediaType = "video"
		ext = ".mp4"
	}

	dlURL := fmt.Sprintf(
		"%s/download?url=%s&type=%s&api_key=%s",
		config.ShrutiAPIURL,
		videoID,
		mediaType,
		config.ShrutiAPIKey,
	)

	path := getPath(track, ext)

	gologging.DebugF("ShrutiApi: Downloading %s (%s)", videoID, mediaType)

	resp, err := rc.R().
		SetContext(ctx).
		SetOutputFileName(path).
		Get(dlURL)
	if err != nil {
		os.Remove(path)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}
		return "", fmt.Errorf("http download failed: %w", err)
	}

	if resp.IsError() {
		os.Remove(path)
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode())
	}

	if !fileExists(path) {
		return "", errors.New("empty file returned by api")
	}

	gologging.InfoF("ShrutiApi: Downloaded %s -> %s", videoID, path)
	return path, nil
}
