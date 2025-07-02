package model

type StoredToken struct {
	UserID    string
	TokenHash string
	UserAgent string
	IPAddress string
	Used      bool
}
