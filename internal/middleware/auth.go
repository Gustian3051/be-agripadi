package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const userIDContextKey = "user_id"

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type jwtClaims struct {
	UserID string `json:"user_id"`
	Sub    string `json:"sub"`
	Exp    int64  `json:"exp"`
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := parseBearerToken(c.GetHeader("Authorization"), secret)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": err.Error()},
			)
			return
		}

		c.Set(userIDContextKey, userID)
		c.Next()
	}
}

func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(userIDContextKey)
	if !exists {
		return uuid.Nil, false
	}

	switch userID := value.(type) {
	case uuid.UUID:
		return userID, userID != uuid.Nil
	case string:
		parsedID, err := uuid.Parse(userID)
		return parsedID, err == nil && parsedID != uuid.Nil
	default:
		return uuid.Nil, false
	}
}

func MustGetUserID(c *gin.Context) uuid.UUID {
	userID, ok := GetUserID(c)
	if !ok {
		panic("user_id is missing from gin context")
	}

	return userID
}

func parseBearerToken(header string, secret string) (uuid.UUID, error) {
	if strings.TrimSpace(secret) == "" {
		return uuid.Nil, errors.New("jwt secret is not configured")
	}

	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(header, bearerPrefix) {
		return uuid.Nil, errors.New("authorization bearer token is required")
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return uuid.Nil, errors.New("invalid token")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return uuid.Nil, errors.New("invalid token header")
	}

	var tokenHeader jwtHeader
	if err := json.Unmarshal(headerBytes, &tokenHeader); err != nil {
		return uuid.Nil, errors.New("invalid token header")
	}

	if tokenHeader.Algorithm != "HS256" {
		return uuid.Nil, errors.New("unsupported token algorithm")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	expectedSignature := mac.Sum(nil)

	actualSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return uuid.Nil, errors.New("invalid token signature")
	}

	if !hmac.Equal(actualSignature, expectedSignature) {
		return uuid.Nil, errors.New("invalid token signature")
	}

	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, errors.New("invalid token claims")
	}

	var claims jwtClaims
	if err := json.Unmarshal(claimBytes, &claims); err != nil {
		return uuid.Nil, errors.New("invalid token claims")
	}

	if claims.Exp > 0 && time.Now().Unix() >= claims.Exp {
		return uuid.Nil, errors.New("token expired")
	}

	rawUserID := strings.TrimSpace(claims.UserID)
	if rawUserID == "" {
		rawUserID = strings.TrimSpace(claims.Sub)
	}

	userID, err := uuid.Parse(rawUserID)
	if err != nil || userID == uuid.Nil {
		return uuid.Nil, errors.New("invalid user_id claim")
	}

	return userID, nil
}
