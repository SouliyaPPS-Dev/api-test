package auth

// TokenManager abstracts token issuance and verification.
type TokenManager interface {
	Generate(userID string) (string, error)
	Validate(token string) (string, error)
	ExtractUserID(token string) (string, error)
}
