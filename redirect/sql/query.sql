-- name: FindByCode :one
SELECT * FROM link
WHERE code = $1 AND is_active = true
LIMIT 1;