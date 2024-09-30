// Package cryptoutil provides utility functions for encryption and decryption.
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt"
)

var (
	ErrTokenExpired   = errors.New("token expired")
	ErrMalformedToken = errors.New("malformed token")
)

// Encrypts data using AES algorithm. The key should be 16, 24, or 32 for 128, 192, or 256 bit encryption respectively.
func EncryptAES(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, errors.Join(errors.New("could not create cipher block"), err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Join(errors.New("could not create GCM"), err)
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, errors.Join(errors.New("could not create nonce"), err)
	}
	//Append cipher to nonce and return nonce + cipher
	return gcm.Seal(nonce, nonce, data, nil), nil
}

// Decrypts data using AES algorithm. The key should be same key that was used to encrypt the data.
func DecryptAES(encryptedData []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, errors.Join(errors.New("could not create cipher block"), err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Join(errors.New("could not create GCM"), err)
	}
	nonceSize := gcm.NonceSize()

	//Get nonce from encrypted data
	nonce, cipher := encryptedData[:nonceSize], encryptedData[nonceSize:]
	data, err := gcm.Open(nil, nonce, cipher, nil)
	if err != nil {
		return nil, errors.Join(errors.New("could not decrypt"), err)
	}
	return data, nil
}

func GenerateJWT(userId string, expiresIn time.Duration, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.StandardClaims{Id: fmt.Sprintf("%v", userId), ExpiresAt: time.Now().Add(expiresIn).Unix()})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("could not get signed token string: %w", err)
	}
	return tokenString, nil
}

func VerifyJWT(token string, secret string) (string, error) {
	claims := new(jwt.StandardClaims)
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		if err, ok := err.(*jwt.ValidationError); ok {
			switch err.Errors {
			case jwt.ValidationErrorExpired:
				return "", ErrTokenExpired
			case jwt.ValidationErrorMalformed:
				return "", ErrMalformedToken
			}
		}
		return "", err
	}

	return claims.Id, nil
}

func GenerateAccessAndRefreshTokens(userId string, accessTokenExpiry time.Duration, refreshTokenExpiry time.Duration, secret string) (string, string) {
	accessToken, _ := GenerateJWT(userId, accessTokenExpiry, secret)
	refreshToken, _ := GenerateJWT(userId, refreshTokenExpiry, secret)
	return accessToken, refreshToken
}

func RandomString() string {
	var buf = make([]byte, 64)
	_, _ = rand.Read(buf)

	var i big.Int
	return i.SetBytes(buf).Text(62)
}

func Base62Hash(text string) string {
	hasher := sha256.New()
	buf := hasher.Sum([]byte(text))

	var i big.Int
	return i.SetBytes(buf).Text(62)
}
