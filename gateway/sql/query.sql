-- name: FindByEmail :one
SELECT id, email, password_hash, is_active, created_at, updated_at FROM users WHERE email = $1 LIMIT 1;

-- name: CreateUser :exec
INSERT INTO users (id, email, password_hash, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6);
