-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (gen_random_uuid(), NOW(), NOW(), $1, $2)
RETURNING id, created_at, updated_at, email, is_chirpy_red;

-- name: DeleteUsers :exec
DELETE FROM users;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEml :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUser :exec
UPDATE users 
SET updated_at = NOW(), email = $1, hashed_password = $2 
WHERE id = $3;

-- name: UpdateRed :exec 
UPDATE users
SET is_chirpy_red = $1
WHERE id = $2;
