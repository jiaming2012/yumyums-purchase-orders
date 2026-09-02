package onboarding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
	"github.com/yumyums/hq/internal/photos"
)

// Video part URL classes, as reported by the finder.
//
//	ok             — on the current bucket prefix; nothing to do
//	dead           — an http(s) URL off the current prefix (stranded on a prior
//	                 host, e.g. the pre-B2 DO Spaces bucket — B-172's class)
//	never_uploaded — empty, blob: or data: — the upload never completed, there
//	                 is nothing to recover; re-upload via the Builder
const (
	videoURLOK            = "ok"
	videoURLDead          = "dead"
	videoURLNeverUploaded = "never_uploaded"
)

// VideoUploader stores an object under key in the current bucket and returns
// its public URL. Production callers use NewSpacesVideoUploader; tests inject
// a stub.
type VideoUploader func(ctx context.Context, key, contentType string, body io.Reader) (string, error)

// NewSpacesVideoUploader adapts the shared S3 client to VideoUploader.
func NewSpacesVideoUploader(client *s3.Client, bucket, endpoint string) VideoUploader {
	if client == nil {
		return nil
	}
	return func(ctx context.Context, key, contentType string, body io.Reader) (string, error) {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        body,
			ContentType: aws.String(contentType),
		})
		if err != nil {
			return "", fmt.Errorf("put %s: %w", key, err)
		}
		return photos.PublicURL(endpoint, bucket, key), nil
	}
}

// videoPartRow is one finder result.
type videoPartRow struct {
	PartID    string `json:"part_id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	ThumbURL  string `json:"thumbnail_url,omitempty"`
	ItemLabel string `json:"item_label"`
	Section   string `json:"section"`
	Template  string `json:"template"`
	Class     string `json:"class"`
}

func classifyVideoURL(u, storagePrefix string) string {
	u = strings.TrimSpace(u)
	switch {
	case u == "" || strings.HasPrefix(u, "blob:") || strings.HasPrefix(u, "data:"):
		return videoURLNeverUploaded
	case storagePrefix != "" && strings.HasPrefix(u, storagePrefix):
		return videoURLOK
	default:
		return videoURLDead
	}
}

func findVideoParts(ctx context.Context, pool *pgxpool.Pool, storagePrefix string) ([]videoPartRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT vp.id, vp.title, vp.url, COALESCE(vp.thumbnail_url, ''),
		       oi.label, os.title, ot.name
		FROM ob_video_parts vp
		JOIN ob_items oi ON oi.id = vp.item_id
		JOIN ob_sections os ON os.id = oi.section_id
		JOIN ob_templates ot ON ot.id = os.template_id
		ORDER BY ot.name, os.sort_order, oi.sort_order, vp.sort_order`)
	if err != nil {
		return nil, fmt.Errorf("list video parts: %w", err)
	}
	defer rows.Close()

	var parts []videoPartRow
	for rows.Next() {
		var p videoPartRow
		if err := rows.Scan(&p.PartID, &p.Title, &p.URL, &p.ThumbURL, &p.ItemLabel, &p.Section, &p.Template); err != nil {
			return nil, fmt.Errorf("scan video part: %w", err)
		}
		p.Class = classifyVideoURL(p.URL, storagePrefix)
		parts = append(parts, p)
	}
	return parts, rows.Err()
}

// recoverKeyFor derives the current-bucket key for a stranded URL. The app's
// own uploads live under "videos/..." — reuse that key when the old URL still
// carries it, so a bulk-copied object lands in the same place; otherwise fall
// back to a recovery prefix keyed by part id.
func recoverKeyFor(rawURL, partID string) string {
	if u, err := url.Parse(rawURL); err == nil {
		p := strings.TrimPrefix(u.Path, "/")
		if idx := strings.Index(p, "videos/"); idx >= 0 {
			if key, err2 := url.PathUnescape(p[idx:]); err2 == nil {
				return key
			}
			return p[idx:]
		}
		if base := path.Base(p); base != "" && base != "." && base != "/" {
			if b, err2 := url.PathUnescape(base); err2 == nil {
				base = b
			}
			return "videos/onboarding/recovered/" + partID + "/" + base
		}
	}
	return "videos/onboarding/recovered/" + partID + "/video"
}

// maxRecoverBytes caps a single fetched object. Phone-recorded training clips
// run tens of MB; anything past this is not one of ours.
const maxRecoverBytes = 512 << 20

// RecoverVideosHandler drives recovery of onboarding video URLs stranded off
// the current bucket — the B-172 class, videos edition. Requires
// admin/manager (route also carries the onboarding grant).
//
// Request body (optional; empty body = real run over every dead part):
//
//	{ "dry_run": true|false, "limit": 0 }
//
// Dry-run responds with the full finder inventory (every part, classified)
// and touches nothing. A real run, synchronously per dead part: fetches the
// old URL over plain HTTPS (the app plays these URLs unauthenticated in a
// <video> tag, so live objects are public reads), re-uploads the bytes to the
// current bucket, and rewrites ob_video_parts.url (and thumbnail_url when it
// is also dead). never_uploaded parts are reported, never fetched — there is
// nothing behind a blob: URL to recover; re-upload those via the Builder.
func RecoverVideosHandler(pool *pgxpool.Pool, storagePrefix string, upload VideoUploader) http.HandlerFunc {
	fetch := &http.Client{Timeout: 5 * time.Minute}
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isManagerOrAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		if storagePrefix == "" || upload == nil {
			writeError(w, http.StatusServiceUnavailable, "storage_unconfigured")
			return
		}

		var req struct {
			DryRun bool `json:"dry_run"`
			Limit  int  `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		parts, err := findVideoParts(r.Context(), pool, storagePrefix)
		if err != nil {
			slog.Error("RecoverVideos finder failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		counts := map[string]int{}
		for _, p := range parts {
			counts[p.Class]++
		}

		if req.DryRun {
			writeJSON(w, http.StatusOK, map[string]any{
				"dry_run":        true,
				"total":          len(parts),
				"ok":             counts[videoURLOK],
				"dead":           counts[videoURLDead],
				"never_uploaded": counts[videoURLNeverUploaded],
				"parts":          parts,
			})
			return
		}

		var recovered, failed []map[string]any
		done := 0
		for _, p := range parts {
			if p.Class != videoURLDead {
				continue
			}
			if req.Limit > 0 && done >= req.Limit {
				break
			}
			done++
			newURL, thumbURL, recErr := recoverOnePart(r.Context(), pool, fetch, upload, p, storagePrefix)
			if recErr != nil {
				slog.Error("RecoverVideos part failed", "part_id", p.PartID, "url", p.URL, "error", recErr)
				failed = append(failed, map[string]any{
					"part_id": p.PartID, "title": p.Title, "template": p.Template,
					"item_label": p.ItemLabel, "url": p.URL, "error": recErr.Error(),
				})
				continue
			}
			recovered = append(recovered, map[string]any{
				"part_id": p.PartID, "title": p.Title, "template": p.Template,
				"item_label": p.ItemLabel, "url": newURL, "thumbnail_url": thumbURL,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"dry_run":        false,
			"total":          len(parts),
			"ok":             counts[videoURLOK],
			"dead":           counts[videoURLDead],
			"never_uploaded": counts[videoURLNeverUploaded],
			"recovered":      recovered,
			"failed":         failed,
		})
	}
}

// fetchObject GETs a stranded URL and returns its bytes + content type.
func fetchObject(ctx context.Context, fetch *http.Client, rawURL, fallbackType string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := fetch.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("old host answered %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRecoverBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(body) > maxRecoverBytes {
		return nil, "", fmt.Errorf("object exceeds %d bytes", maxRecoverBytes)
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" || ct == "application/octet-stream" {
		if byExt := mime.TypeByExtension(path.Ext(rawURL)); byExt != "" {
			ct = byExt
		} else if fallbackType != "" {
			ct = fallbackType
		} else {
			ct = "application/octet-stream"
		}
	}
	return body, ct, nil
}

func recoverOnePart(ctx context.Context, pool *pgxpool.Pool, fetch *http.Client, upload VideoUploader, p videoPartRow, storagePrefix string) (newURL, newThumb string, err error) {
	body, ct, err := fetchObject(ctx, fetch, p.URL, "video/mp4")
	if err != nil {
		return "", "", fmt.Errorf("fetch: %w", err)
	}
	newURL, err = upload(ctx, recoverKeyFor(p.URL, p.PartID), ct, bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("upload: %w", err)
	}

	// The thumbnail rides along when it is also stranded; losing it is not
	// worth failing the video over — a missing poster degrades gracefully.
	newThumb = p.ThumbURL
	if classifyVideoURL(p.ThumbURL, storagePrefix) == videoURLDead {
		if tBody, tCT, tErr := fetchObject(ctx, fetch, p.ThumbURL, "image/jpeg"); tErr == nil {
			if u, upErr := upload(ctx, recoverKeyFor(p.ThumbURL, p.PartID+"-thumb"), tCT, bytes.NewReader(tBody)); upErr == nil {
				newThumb = u
			}
		}
	}

	_, err = pool.Exec(ctx,
		`UPDATE ob_video_parts SET url = $1, thumbnail_url = NULLIF($2, '') WHERE id = $3`,
		newURL, newThumb, p.PartID)
	if err != nil {
		return "", "", fmt.Errorf("rewrite: %w", err)
	}
	return newURL, newThumb, nil
}
