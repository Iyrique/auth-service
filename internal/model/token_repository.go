package model

type TokenRepository interface {
	StoreRefreshToken(userID, tokenHash, userAgent, ip string) error
	FindActiveToken(userID string) (*StoredToken, error)
	InvalidateToken(userID string) error
}
