-- name: FindAllByUserID :many

SELECT * FROM link 
WHERE user_id = $1 AND is_active = true 
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: FindByCodeAndUserID :one

SELECT * FROM link 
WHERE code = $1 AND user_id = $2 AND is_active = true
LIMIT 1;

-- name: CreateLink :one

INSERT INTO link (
    user_id, code, original_url, custom_alias, expires_at, is_active, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, user_id, code, original_url, custom_alias, expires_at, is_active, created_at;;

-- name: UpdateLink :execrows

UPDATE link
SET 
    code = $2,
    original_url = $3,
    custom_alias = $4,
    expires_at = $5,
    is_active = $6
WHERE id = $1;