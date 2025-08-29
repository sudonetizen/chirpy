module github.com/sudonetizen/chirpy

go 1.24.2

replace github.com/sudonetizen/database v0.0.0 => ./internal/database/

require github.com/sudonetizen/database v0.0.0 

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
)
