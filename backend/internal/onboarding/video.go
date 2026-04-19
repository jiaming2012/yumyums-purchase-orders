package onboarding

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/photos"
)

// convertToMP4 converts a video file to MP4 using FFmpeg with H.264/AAC encoding.
// Uses exec.CommandContext with a 5 minute timeout.
func convertToMP4(ctx context.Context, inputPath, outputPath string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-movflags", "+faststart",
		"-y", outputPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg convert: %w — output: %s", err, string(out))
	}
	return nil
}

// extractThumbnail extracts a single frame from a video as a JPEG thumbnail.
// Seeks to 2 seconds in and captures the first frame.
func extractThumbnail(ctx context.Context, inputPath, thumbPath string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-i", inputPath,
		"-ss", "00:00:02",
		"-frames:v", "1",
		"-q:v", "3",
		"-y", thumbPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg thumbnail: %w — output: %s", err, string(out))
	}
	return nil
}

// downloadFromSpaces downloads an object from DO Spaces to a local file.
// Generates a presigned GET URL (1 hour TTL) and fetches it via HTTP.
func downloadFromSpaces(ctx context.Context, presigner *s3.PresignClient, bucket, key, destPath string) error {
	getURL, err := photos.GeneratePresignedGetURL(ctx, presigner, bucket, key, time.Hour)
	if err != nil {
		return fmt.Errorf("presign get %s: %w", key, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getURL, nil)
	if err != nil {
		return fmt.Errorf("create GET request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned status %d", key, resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create dest file %s: %w", destPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write dest file: %w", err)
	}
	return nil
}

// uploadToSpaces uploads a local file to DO Spaces using a presigned PUT URL.
func uploadToSpaces(ctx context.Context, presigner *s3.PresignClient, bucket, key, srcPath, contentType string) error {
	putURL, err := photos.GeneratePresignedPutURL(ctx, presigner, bucket, key, contentType, 15*time.Minute)
	if err != nil {
		return fmt.Errorf("presign put %s: %w", key, err)
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src file %s: %w", srcPath, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat src file: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, putURL, f)
	if err != nil {
		return fmt.Errorf("create PUT request: %w", err)
	}
	req.ContentLength = stat.Size()
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("PUT %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("PUT %s returned status %d", key, resp.StatusCode)
	}
	return nil
}

// processVideo handles FFmpeg conversion and thumbnail extraction for a video part.
// Downloads raw file, converts if needed (.mov/.webm -> .mp4), extracts thumbnail,
// uploads both back to Spaces, and updates ob_video_parts in the DB.
func processVideo(ctx context.Context, presigner *s3.PresignClient, bucket, endpoint string, pool *pgxpool.Pool, partID, objectKey string) error {
	tmpDir, err := os.MkdirTemp("", "yumyums-video-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Determine input extension
	ext := strings.ToLower(filepath.Ext(objectKey))
	inputPath := filepath.Join(tmpDir, "input"+ext)

	// Download raw file from Spaces
	if err := downloadFromSpaces(ctx, presigner, bucket, objectKey, inputPath); err != nil {
		return fmt.Errorf("download from spaces: %w", err)
	}

	// Key without extension (for derived keys)
	keyWithoutExt := strings.TrimSuffix(objectKey, filepath.Ext(objectKey))

	// Convert to MP4 if needed
	videoPath := inputPath
	videoKey := objectKey
	if ext == ".mov" || ext == ".webm" {
		outputPath := filepath.Join(tmpDir, "output.mp4")
		if err := convertToMP4(ctx, inputPath, outputPath); err != nil {
			return fmt.Errorf("convert to mp4: %w", err)
		}

		mp4Key := keyWithoutExt + ".mp4"
		if err := uploadToSpaces(ctx, presigner, bucket, mp4Key, outputPath, "video/mp4"); err != nil {
			return fmt.Errorf("upload mp4: %w", err)
		}

		mp4PublicURL := photos.PublicURL(endpoint, bucket, mp4Key)
		if _, err := pool.Exec(ctx, `UPDATE ob_video_parts SET url = $1 WHERE id = $2`, mp4PublicURL, partID); err != nil {
			return fmt.Errorf("update video url: %w", err)
		}

		videoPath = outputPath
		videoKey = mp4Key
	}
	_ = videoKey // used above for mp4 path tracking

	// Extract thumbnail
	thumbPath := filepath.Join(tmpDir, "thumb.jpg")
	if err := extractThumbnail(ctx, videoPath, thumbPath); err != nil {
		return fmt.Errorf("extract thumbnail: %w", err)
	}

	thumbKey := keyWithoutExt + "_thumb.jpg"
	if err := uploadToSpaces(ctx, presigner, bucket, thumbKey, thumbPath, "image/jpeg"); err != nil {
		return fmt.Errorf("upload thumbnail: %w", err)
	}

	thumbPublicURL := photos.PublicURL(endpoint, bucket, thumbKey)
	if _, err := pool.Exec(ctx, `UPDATE ob_video_parts SET thumbnail_url = $1 WHERE id = $2`, thumbPublicURL, partID); err != nil {
		return fmt.Errorf("update thumbnail url: %w", err)
	}

	log.Printf("processVideo: completed part %s — thumb %s", partID, thumbPublicURL)
	return nil
}
