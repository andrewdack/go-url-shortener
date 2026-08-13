package store

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrAlreadyExists = errors.New("short_url already exists")

// insertUrl inserts a new short_url -> long_url mapping owned by userId.
func insertUrl(shortUrl, longUrl, userId string) error {
	_, err := storeService.pgPool.Exec(ctx,
		`INSERT INTO urls (short_url, long_url, user_id) VALUES ($1, $2, $3)`,
		shortUrl, longUrl, userId,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return fmt.Errorf("failed inserting url %s: %w", shortUrl, err)
	}
	return nil
}

// getUrlByShortCode returns the long URL and owning user ID for a short code.
func getUrlByShortCode(shortUrl string) (longUrl string, userId string, err error) {
	err = storeService.pgPool.QueryRow(ctx,
		`SELECT long_url, user_id FROM urls WHERE short_url = $1`,
		shortUrl,
	).Scan(&longUrl, &userId)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("failed querying url %s: %w", shortUrl, err)
	}
	return longUrl, userId, nil
}
