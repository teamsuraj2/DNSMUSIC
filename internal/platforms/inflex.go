package platforms

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"

	state "main/internal/core/models"
)

const (
	PlatformInflexApi state.PlatformName = "InflexApi"
)

var (
	inflexApiKey = os.Getenv("INFLEX_API_KEY")
	inflexApiURL = "https://teaminflex.xyz"
)

type inflexDownloadResponse struct {
	Status      string `json:"status"`
	Detail      string `json:"detail"`
	DownloadURL string `json:"download_url"`
}

type InflexApiPlatform struct {
	name state.PlatformName
}

func init() {
	Register(81, &InflexApiPlatform{
		name: PlatformInflexApi,
	})
}

func (f *InflexApiPlatform) Name() state.PlatformName {
	return f.name
}

func (f *InflexApiPlatform) CanGetTracks(query string) bool {
	return false
}

func (f *InflexApiPlatform) GetTracks(_ string, _ bool) ([]*state.Track, error) {
	return nil, errors.New("inflexapi is a download-only platform")
}

func (f *InflexApiPlatform) CanDownload(source state.PlatformName) bool {
	if inflexApiKey == "" {
		return false
	}
	return source == PlatformYouTube
}

func (f *InflexApiPlatform) Download(
	ctx context.Context,
	track *state.Track,
	_ *telegram.NewMessage,
) (string, error) {
	if cached := findFile(track); cached != "" {
		gologging.Debug("InflexApi: Download -> Cached File -> " + cached)
		return cached, nil
	}

	return f.download(ctx, track)
}

func (*InflexApiPlatform) CanSearch() bool { return false }

func (*InflexApiPlatform) Search(string, bool) ([]*state.Track, error) {
	return nil, nil
}

func (f *InflexApiPlatform) download(ctx context.Context, track *state.Track) (string, error) {
	videoID := track.ID
	if videoID == "" {
		videoID = track.URL
	}

	mediaType := "audio"
	ext := ".webm"
	if track.Video {
		mediaType = "video"
		ext = ".mkv"
	}

	path := getPath(track, ext)

	gologging.DebugF("InflexApi: Requesting %s (%s)", videoID, mediaType)

	payload := map[string]string{
		"url":  videoID,
		"type": mediaType,
	}

	var result inflexDownloadResponse

	resp, err := rc.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-API-KEY", inflexApiKey).
		SetBody(payload).
		SetResult(&result).
		Post(inflexApiURL + "/download")
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}
		return "", fmt.Errorf("inflexapi request failed: %w", err)
	}

	if resp.IsError() {
		return "", fmt.Errorf("inflexapi returned status: %d", resp.StatusCode())
	}

	if result.Status == "error" {
		detail := result.Detail
		if detail == "" {
			detail = "unknown error"
		}
		return "", fmt.Errorf("inflexapi error: %s", detail)
	}

	if result.Status != "success" || result.DownloadURL == "" {
		return "", errors.New("inflexapi: unexpected response")
	}

	fileResp, err := rc.R().
		SetContext(ctx).
		SetOutputFileName(path).
		Get(inflexApiURL + result.DownloadURL)
	if err != nil {
		os.Remove(path)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}
		return "", fmt.Errorf("inflexapi download failed: %w", err)
	}

	if fileResp.IsError() {
		os.Remove(path)
		return "", fmt.Errorf("inflexapi file download failed with status: %d", fileResp.StatusCode())
	}

	if !fileExists(path) {
		return "", errors.New("empty file returned by api")
	}

	gologging.InfoF("InflexApi: Downloaded %s -> %s", videoID, path)
	return path, nil
}