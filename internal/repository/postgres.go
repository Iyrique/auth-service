package repository

import (
	"auth-service/internal/model"
	"database/sql"
	"errors"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) StoreRefreshToken(userID, tokenHash, userAgent, ip string) error {
	_, err := r.db.Exec(`
		INSERT INTO refresh_tokens (id, user_id, token_hash, user_agent, ip_address)
		VALUES (gen_random_uuid(), $1, $2, $3, $4)
	`, userID, tokenHash, userAgent, ip)
	return err
}

func (r *PostgresRepository) FindActiveToken(userID string) (*model.StoredToken, error) {
	row := r.db.QueryRow(`
		SELECT token_hash, user_agent, ip_address, used
		FROM refresh_tokens
		WHERE user_id = $1
		ORDER BY issued_at DESC LIMIT 1
	`, userID)

	var tok model.StoredToken
	tok.UserID = userID
	if err := row.Scan(&tok.TokenHash, &tok.UserAgent, &tok.IPAddress, &tok.Used); err != nil {
		return nil, errors.New("refresh token not found")
	}
	return &tok, nil
}

func (r *PostgresRepository) InvalidateToken(userID string) error {
	_, err := r.db.Exec(`
		UPDATE refresh_tokens SET used = true WHERE user_id = $1
	`, userID)
	return err
}
