package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leirbagxis/FreddyBot/pkg/config"
)

var secreteKey = []byte(config.SecreteKey)

type Role string

const (
	RoleOwner Role = "owner"
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type Claims struct {
	UserID int64 `json:"user_id"`
	Role   Role  `json:"role"`
	TV     int64 `json:"tv"` // token version para o usuário (se quisermos invalidar todos os tokens de um user)
	jwt.RegisteredClaims
}

// GetUserID extrai o userID do contexto JWT do Gin.
// Retorna 0 se nao estiver disponivel.
func GetUserID(ctx *gin.Context) int64 {
	userIDRaw, exists := ctx.Get("userID")
	if !exists {
		return 0
	}
	userID, ok := userIDRaw.(int64)
	if !ok {
		return 0
	}
	return userID
}

func GenerateToken(userID int64, role Role, tv int64, ttl time.Duration) (string, error) {
	if userID == 0 {
		return "", errors.New("userID is required")
	}
	if ttl <= 0 {
		return "", errors.New("ttl must be > 0")
	}
	if len(secreteKey) < 32 {
		return "", fmt.Errorf("JWT secret too short")
	}

	now := time.Now()

	claims := Claims{
		UserID: userID,
		Role:   role,
		TV:     tv,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    config.JWTIssuer,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-10 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        newJTI(),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secreteKey)
}

func ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secreteKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.Issuer != config.JWTIssuer {
		return nil, errors.New("invalid issuer")
	}

	return claims, nil
}

func newJTI() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		b2 := make([]byte, 16)
		for i := range b2 {
			b2[i] = byte(time.Now().UnixNano() & 0xff)
			time.Sleep(time.Nanosecond)
		}
		return fmt.Sprintf("%x", b2)
	}
	return fmt.Sprintf("%x", b)
}
