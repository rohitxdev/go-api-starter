package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateLoginToken(userId string, jwtSecret string) (string, error) {
	claims := jwt.MapClaims{
		"userId": userId,
		"nbf":    time.Now().Unix(),
		"exp":    time.Now().Add(5 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("could not generate login token: %w", err)
	}
	return tokenString, nil
}

func ValidateLoginToken(tokenStr string, jwtSecret string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return "", fmt.Errorf("could not validate login token: %w", err)
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		userId, ok := claims["userId"].(string)
		if !ok {
			return "", fmt.Errorf("could not validate login token: userId is not a string")
		}
		return userId, nil
	}
	return "", fmt.Errorf("could not validate login token: invalid claims")
}
