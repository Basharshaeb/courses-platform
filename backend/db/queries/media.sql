-- ---- Media Assets ----

-- name: CreateMediaAsset :one
INSERT INTO media_assets (
  id, school_id, provider, provider_ref, kind, status, filename, size_bytes, duration_sec, thumbnail_url
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetMediaAsset :one
SELECT * FROM media_assets WHERE school_id = $1 AND id = $2;

-- name: GetMediaAssetByProviderRef :one
SELECT * FROM media_assets
WHERE school_id = $1 AND provider = $2 AND provider_ref = $3;

-- name: UpdateMediaAssetStatus :one
UPDATE media_assets SET
  status        = sqlc.arg(status),
  duration_sec  = coalesce(sqlc.narg(duration_sec), duration_sec),
  thumbnail_url = coalesce(sqlc.narg(thumbnail_url), thumbnail_url)
WHERE school_id = sqlc.arg(school_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: ListMediaAssets :many
SELECT * FROM media_assets
WHERE school_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
