package middleware

import (
	"crypto/rsa"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type AuthMiddleware struct {
	publicKey *rsa.PublicKey
	redis     *redis.Client
}

func NewAuthMiddleware(publicKeyPath string) *AuthMiddleware {
	keyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		panic("failed to read JWT public key: " + err.Error())
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		panic("failed to parse JWT public key: " + err.Error())
	}

	return &AuthMiddleware{publicKey: pubKey}
}

func (m *AuthMiddleware) WithRedis(rdb *redis.Client) *AuthMiddleware {
	m.redis = rdb
	return m
}

func (m *AuthMiddleware) Handle() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization format",
			})
		}

		tokenStr := parts[1]

		// Check token blacklist
		if m.redis != nil {
			val, _ := m.redis.Get(c.Context(), "token_blacklist:"+tokenStr).Result()
			if val != "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Token has been revoked",
				})
			}
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok || t.Method.Alg() != "RS256" {
				return nil, jwt.ErrSignatureInvalid
			}
			return m.publicKey, nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}

		sub, _ := claims["sub"].(string)
		tenantID, _ := claims["tenant_id"].(string)
		role, _ := claims["role"].(string)

		if sub == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token: missing subject",
			})
		}

		// Store token expiry for logout blacklisting
		if exp, ok := claims["exp"].(float64); ok {
			c.Locals("token_exp", time.Unix(int64(exp), 0))
		}
		c.Locals("token_raw", tokenStr)
		c.Locals("user_id", sub)
		c.Locals("tenant_id", tenantID)
		c.Locals("role", role)

		return c.Next()
	}
}
