module github.com/sudonetizen/chirpy

go 1.24.2

replace github.com/sudonetizen/database v0.0.0 => ./internal/database/

replace github.com/sudonetizen/auth v0.0.0 => ./internal/auth/

require github.com/sudonetizen/database v0.0.0

require (
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/sudonetizen/auth v0.0.0
)

require (
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	golang.org/x/crypto v0.41.0 // indirect
)
