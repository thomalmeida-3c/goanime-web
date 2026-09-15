package upscaler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultVideoConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultVideoConfig()
	assert.Empty(t, cfg.InputPath)
	assert.NotZero(t, cfg.Anime4KOptions.ScaleFactor)
}

func TestNewVideoUpscaler_MissingFFmpeg(t *testing.T) {
	t.Parallel()
	// NewVideoUpscaler validates inputs and tries to find ffmpeg.
	// In CI without ffmpeg this returns an error; in dev with ffmpeg it may
	// succeed. We assert it does not panic on empty config.
	cfg := DefaultVideoConfig()
	_, err := NewVideoUpscaler(cfg)
	_ = err
}

func TestValidateFFmpeg_DoesNotPanic(t *testing.T) {
	t.Parallel()
	_, err := ValidateFFmpeg()
	_ = err
}

func TestUpscaleVideoFile_MissingInput(t *testing.T) {
	t.Parallel()
	err := UpscaleVideoFile("/no/input.mp4", "/tmp/out.mp4", DefaultOptions())
	assert.Error(t, err)
}

func TestGetVideoInfo_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := GetVideoInfo("/no/video.mp4")
	assert.Error(t, err)
}

func TestEstimateUpscaleTime_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := EstimateUpscaleTime("/no/video.mp4", DefaultOptions())
	assert.Error(t, err)
}

func TestVideoUpscaler_UpscaleVideo_CanceledContext(t *testing.T) {
	t.Parallel()
	cfg := DefaultVideoConfig()
	cfg.InputPath = "/no/input.mp4"
	cfg.OutputPath = "/tmp/out.mp4"
	v, err := NewVideoUpscaler(cfg)
	if err != nil {
		t.Skip("ffmpeg not available")
	}
	defer v.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = v.UpscaleVideo(ctx)
	assert.Error(t, err)
}
