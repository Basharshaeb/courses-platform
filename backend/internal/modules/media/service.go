package media

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/config"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/store"
)

type Service struct {
	pool *pgxpool.Pool
	cfg  config.Config
}

func NewService(pool *pgxpool.Pool, cfg config.Config) *Service {
	return &Service{pool: pool, cfg: cfg}
}

type UploadInit struct {
	AssetID   uuid.UUID         `json:"assetId"`
	Provider  string            `json:"provider"`
	UploadURL string            `json:"uploadUrl"`
	Headers   map[string]string `json:"headers"`
	VideoID   string            `json:"videoId,omitempty"`
}

// PrepareUpload creates a media asset record and returns direct upload credentials.
func (s *Service) PrepareUpload(ctx context.Context, schoolID uuid.UUID, filename string, kind string) (*UploadInit, error) {
	if kind != "video" && kind != "file" && kind != "image" {
		return nil, apperr.Validation("invalid_kind", "kind must be video, file, or image")
	}

	assetID := uuid.Must(uuid.NewV7())
	provider := "s3"
	if kind == "video" && s.cfg.BunnyLibraryID != "" {
		provider = "bunny"
	}

	providerRef := assetID.String()
	headers := make(map[string]string)
	uploadURL := ""

	if provider == "bunny" {
		// Bunny Stream direct TUS upload
		expires := time.Now().Add(2 * time.Hour).Unix()
		expiresStr := strconv.FormatInt(expires, 10)
		signatureRaw := s.cfg.BunnyLibraryID + s.cfg.BunnyAPIKey + expiresStr + providerRef
		h := sha256.New()
		h.Write([]byte(signatureRaw))
		signature := hex.EncodeToString(h.Sum(nil))

		uploadURL = "https://video.bunnycdn.com/tusupload"
		headers["LibraryId"] = s.cfg.BunnyLibraryID
		headers["VideoId"] = providerRef
		headers["AuthorizationSignature"] = signature
		headers["AuthorizationExpire"] = expiresStr
	} else {
		// Generic or mock direct upload endpoint
		uploadURL = fmt.Sprintf("/api/v1/media/upload/%s", assetID)
	}

	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		_, err := q.CreateMediaAsset(ctx, store.CreateMediaAssetParams{
			ID:          assetID,
			SchoolID:    schoolID,
			Provider:    provider,
			ProviderRef: providerRef,
			Kind:        kind,
			Status:      "uploading",
			Filename:    filename,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("media: create asset: %w", err)
	}

	return &UploadInit{
		AssetID:   assetID,
		Provider:  provider,
		UploadURL: uploadURL,
		Headers:   headers,
		VideoID:   providerRef,
	}, nil
}

// GetPlaybackURL generates a token-authenticated streaming URL for a video asset.
func (s *Service) GetPlaybackURL(ctx context.Context, schoolID, assetID uuid.UUID) (string, error) {
	var asset store.MediaAsset
	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		asset, err = q.GetMediaAsset(ctx, store.GetMediaAssetParams{
			SchoolID: schoolID,
			ID:       assetID,
		})
		return err
	})
	if db.IsNotFound(err) {
		return "", apperr.NotFound("media_not_found", "media asset not found")
	}
	if err != nil {
		return "", fmt.Errorf("media: get asset: %w", err)
	}

	if asset.Provider == "bunny" && s.cfg.BunnyPullZone != "" && s.cfg.BunnyTokenKey != "" {
		expires := time.Now().Add(4 * time.Hour).Unix()
		expiresStr := strconv.FormatInt(expires, 10)
		path := fmt.Sprintf("/%s/playlist.m3u8", asset.ProviderRef)
		hashable := s.cfg.BunnyTokenKey + path + expiresStr

		h := sha256.New()
		h.Write([]byte(hashable))
		token := base64.URLEncoding.EncodeToString(h.Sum(nil))
		token = strings.TrimRight(token, "=")

		return fmt.Sprintf("https://%s%s?token=%s&expires=%s", s.cfg.BunnyPullZone, path, token, expiresStr), nil
	}

	// Fallback mock stream URL for local development
	return fmt.Sprintf("/api/v1/media/stream/%s", asset.ID), nil
}

// HandleBunnyWebhook updates the video encoding status from Bunny Stream.
func (s *Service) HandleBunnyWebhook(ctx context.Context, videoID string, status int, durationSec int32, thumbnailURL string) error {
	var mediaStatus string
	switch status {
	case 3: // Ready
		mediaStatus = "ready"
	case 4: // Failed
		mediaStatus = "failed"
	default:
		mediaStatus = "processing"
	}

	var durPtr *int32
	if durationSec > 0 {
		durPtr = &durationSec
	}
	var thumbPtr *string
	if thumbnailURL != "" {
		thumbPtr = &thumbnailURL
	}

	// We query the asset by provider_ref
	_, err := s.pool.Exec(ctx, `
		UPDATE media_assets
		SET status = $1, duration_sec = coalesce($2, duration_sec), thumbnail_url = coalesce($3, thumbnail_url), updated_at = now()
		WHERE provider = 'bunny' AND provider_ref = $4
	`, mediaStatus, durPtr, thumbPtr, videoID)
	return err
}
