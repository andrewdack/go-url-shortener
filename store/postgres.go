package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// insertUrl inserts a new short_url -> long_url mapping owned by userId.
func (s *StorageService) insertUrl(ctx context.Context, shortUrl, longUrl, userId string) error {
	_, err := s.pgPool.Exec(ctx,
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
func (s *StorageService) getUrlByShortCode(ctx context.Context, shortUrl string) (longUrl string, userId string, err error) {
	err = s.pgPool.QueryRow(ctx,
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

// updateUrl updates the long URL for an existing short_url row.
func (s *StorageService) updateUrl(ctx context.Context, shortUrl, newLongUrl string) error {
	tag, err := s.pgPool.Exec(ctx,
		`UPDATE urls SET long_url = $1, updated_at = now() WHERE short_url = $2`,
		newLongUrl, shortUrl,
	)
	if err != nil {
		return fmt.Errorf("failed updating url %s: %w", shortUrl, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// deleteUrl deletes the row for shortUrl.
func (s *StorageService) deleteUrl(ctx context.Context, shortUrl string) error {
	tag, err := s.pgPool.Exec(ctx, `DELETE FROM urls WHERE short_url = $1`, shortUrl)
	if err != nil {
		return fmt.Errorf("failed deleting url %s: %w", shortUrl, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
