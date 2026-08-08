package platforms

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"

	state "main/internal/core/models"
)

const (
	PlatformRiteshApi state.PlatformName = "RiteshApi"
)

var (
	riteshApiKey = os.Getenv("RITESH_API_KEY")
	riteshApiURL = envOrDefault("RITESH_API_URL", "http://yt.riteshyt.in")
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type RiteshApiPlatform struct {
	name state.PlatformName
}

func init() {
	Register(80, &RiteshApiPlatform{
		name: PlatformRiteshApi,
	})
}

func (f *RiteshApiPlatform) Name() state.PlatformName {
	return f.name
}

func (f *RiteshApiPlatform) CanGetTracks(query string) bool {
	return false
}

func (f *RiteshApiPlatform) GetTracks(_ string, _ bool) ([]*state.Track, error) {
	return nil, errors.New("riteshapi is a download-only platform")
}

func (f *RiteshApiPlatform) CanDownload(source state.PlatformName) bool {
	if riteshApiKey == "" {
		return false
	}
	return source == PlatformYouTube
}

func (f *RiteshApiPlatform) Download(
	ctx context.Context,
	track *state.Track,
	_ *telegram.NewMessage,
) (string, error) {
	if cached := findFile(track); cached != "" {
		gologging.Debug("RiteshApi: Download -> Cached File -> " + cached)
		return cached, nil
	}

	return f.download(ctx, track)
}

func (*RiteshApiPlatform) CanSearch() bool { return false }

func (*RiteshApiPlatform) Search(string, bool) ([]*state.Track, error) {
	return nil, nil
}

func (f *RiteshApiPlatform) download(ctx context.Context, track *state.Track) (string, error) {
	videoID := track.ID
	if videoID == "" {
		videoID = track.URL
	}

	dlType := "audio"
	ext := "webm"
	if track.Video {
		dlType = "video"
		ext = "mp4"
	}

	path := getPath(track, "."+ext)

	gologging.DebugF("RiteshApi: Requesting %s (%s)", videoID, dlType)

	// Note: yeh /download endpoint 307 redirect deta hai asli stream URL ki taraf,
	// isliye redirect-following client aavashyak hai.
	dlURL := fmt.Sprintf(
		"%s/download?query=%s&dl_type=%s&api_key=%s",
		riteshApiURL,
		url.QueryEscape(videoID),
		dlType,
		url.QueryEscape(riteshApiKey),
	)

	fileResp, err := rc.R().
		SetContext(ctx).
		SetOutputFileName(path).
		Get(dlURL)
	if err != nil {
		os.Remove(path)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}
		return "", fmt.Errorf("riteshapi download failed: %w", err)
	}

	if fileResp.IsError() {
		os.Remove(path)
		return "", fmt.Errorf("riteshapi file download failed with status: %d", fileResp.StatusCode())
	}

	if !fileExists(path) {
		return "", errors.New("empty file returned by api")
	}

	gologging.InfoF("RiteshApi: Downloaded %s -> %s", videoID, path)
	return path, nil
}
