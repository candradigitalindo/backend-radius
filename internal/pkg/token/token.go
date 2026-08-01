package token

import (
	"crypto/rsa"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/candrasyahputra/radius-server/internal/config"
)

type Manager struct {
	privateKey    *rsa.PrivateKey
	publicKey     *rsa.PublicKey
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

type Claims struct {
	UserID   string `json:"sub"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	Type     string `json:"type"`
	// ImpersonatorID, when set, marks this token as a superadmin session
	// temporarily acting as a tenant user. Empty for normal sessions.
	ImpersonatorID string `json:"-"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func NewManager(cfg config.JWTConfig) *Manager {
	privData, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		panic("failed to read JWT private key: " + err.Error())
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privData)
	if err != nil {
		panic("failed to parse JWT private key: " + err.Error())
	}

	pubData, err := os.ReadFile(cfg.PublicKeyPath)
	if err != nil {
		panic("failed to read JWT public key: " + err.Error())
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubData)
	if err != nil {
		panic("failed to parse JWT public key: " + err.Error())
	}

	return &Manager{
		privateKey:    privKey,
		publicKey:     pubKey,
		accessExpiry:  cfg.AccessExpiry,
		refreshExpiry: cfg.RefreshExpiry,
	}
}

func (m *Manager) GeneratePair(claims Claims) (*TokenPair, error) {
	now := time.Now()

	accessClaims := jwt.MapClaims{
		"sub":       claims.UserID,
		"tenant_id": claims.TenantID,
		"role":      claims.Role,
		"type":      "access",
		"iat":       now.Unix(),
		"exp":       now.Add(m.accessExpiry).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessStr, err := accessToken.SignedString(m.privateKey)
	if err != nil {
		return nil, err
	}

	refreshClaims := jwt.MapClaims{
		"sub":       claims.UserID,
		"tenant_id": claims.TenantID,
		"role":      claims.Role,
		"type":      "refresh",
		"iat":       now.Unix(),
		"exp":       now.Add(m.refreshExpiry).Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(m.privateKey)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresIn:    int64(m.accessExpiry.Seconds()),
	}, nil
}

// GenerateImpersonationToken issues a single short-lived access token (no
// refresh token) for a superadmin temporarily viewing the app as a tenant
// user. The impersonator's user id is embedded so it can be audited.
func (m *Manager) GenerateImpersonationToken(claims Claims, expiry time.Duration) (string, int64, error) {
	now := time.Now()

	accessClaims := jwt.MapClaims{
		"sub":             claims.UserID,
		"tenant_id":       claims.TenantID,
		"role":            claims.Role,
		"type":            "access",
		"impersonator_id": claims.ImpersonatorID,
		"iat":             now.Unix(),
		"exp":             now.Add(expiry).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessStr, err := accessToken.SignedString(m.privateKey)
	if err != nil {
		return "", 0, err
	}

	return accessStr, int64(expiry.Seconds()), nil
}

func (m *Manager) ValidateRefreshToken(tokenStr string) (*Claims, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok || t.Method.Alg() != "RS256" {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.publicKey, nil
	})
	if err != nil || !tok.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	mapClaims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrSignatureInvalid
	}

	tokenType, _ := mapClaims["type"].(string)
	if tokenType != "refresh" {
		return nil, jwt.ErrSignatureInvalid
	}

	userID, ok := mapClaims["sub"].(string)
	if !ok || userID == "" {
		return nil, jwt.ErrSignatureInvalid
	}
	tenantID, ok := mapClaims["tenant_id"].(string)
	if !ok {
		return nil, jwt.ErrSignatureInvalid
	}
	role, _ := mapClaims["role"].(string)

	return &Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		Type:     tokenType,
	}, nil
}
