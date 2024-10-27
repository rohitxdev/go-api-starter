package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateLoginToken generates a login token for the user id.
func GenerateLoginToken(userId uint64, jwtSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"userId": userId,
		"nbf":    time.Now().Unix(),
		"exp":    time.Now().Add(expiresIn).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("Failed to generate login token: %w", err)
	}
	return tokenStr, nil
}

// ValidateLoginToken validates the login token and returns the user id.
func ValidateLoginToken(tokenStr string, jwtSecret string) (uint64, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return 0, fmt.Errorf("Failed to validate login token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("Failed to validate login token: invalid claims")
	}

	userId, ok := claims["userId"].(uint64)
	if !ok {
		return 0, fmt.Errorf("Failed to validate login token: userId is not of type uint64")
	}
	return userId, nil
}
