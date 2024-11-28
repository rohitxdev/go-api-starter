// Package cryptoutil provides utility functions for encryption and decryption.
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/argon2"
)

func padKey(key []byte) []byte {
	keyLen := len(key)
	padDiff := keyLen % 16
	if padDiff == 0 {
		return key
	}
	padLen := 16 - padDiff
	pad := make([]byte, padLen)
	for i := 0; i < padLen; i++ {
		pad[i] = byte(padLen)
	}
	return append(key, pad...)
}

// Encrypts data using AES algorithm. The key should be 16, 24, or 32 for 128, 192, or 256 bit encryption respectively.
func EncryptAES(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(padKey(key))
	if err != nil {
		return nil, fmt.Errorf("could not create cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("could not create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, fmt.Errorf("could not create nonce: %w", err)
	}
	//Append cipher to nonce and return nonce + cipher
	return gcm.Seal(nonce, nonce, data, nil), nil
}

// Decrypts data using AES algorithm. The key should be same key that was used to encrypt the data.
func DecryptAES(encryptedData []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(padKey(key))
	if err != nil {
		return nil, fmt.Errorf("could not create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("could not create GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()

	//Get nonce from encrypted data
	nonce, cipher := encryptedData[:nonceSize], encryptedData[nonceSize:]
	data, err := gcm.Open(nil, nonce, cipher, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt: %w", err)
	}
	return data, nil
}

func RandomString(size uint) string {
	var buf = make([]byte, size)
	_, _ = rand.Read(buf)
	return bufToBase62(buf)
}

func bufToBase62(buf []byte) string {
	var i big.Int
	i.SetBytes(buf)
	return i.Text(62)
}

func Base62Hash(text string) string {
	hasher := sha256.New()
	buf := hasher.Sum([]byte(text))
	return bufToBase62(buf)
}

func Base32Hash(text string) string {
	hasher := sha256.New()
	buf := hasher.Sum([]byte(text))
	return base32.StdEncoding.EncodeToString(buf)
}

func generateSalt(length int) ([]byte, error) {
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

func HashPassword(password string) (string, error) {
	const (
		time    = 1         // number of iterations
		memory  = 64 * 1024 // memory in KiB
		threads = 4         // parallelism
		keyLen  = 32        // length of the generated key
	)

	salt, err := generateSalt(16)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
	fullHash := append(salt, hash...)
	encodedHash := base64.RawStdEncoding.EncodeToString(fullHash)

	return encodedHash, nil
}

func VerifyPassword(password string, fullHash string) bool {
	data, err := base64.RawStdEncoding.DecodeString(fullHash)
	if err != nil {
		return false
	}

	salt := data[:16]
	hash := data[16:]
	newHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	return subtle.ConstantTimeCompare(hash, newHash) == 1
}

func GenerateJWT(userID uint64, expiresIn time.Duration, secret string) (string, error) {
	claims := jwt.MapClaims{
		"id":  userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(expiresIn).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func VerifyJWT(tokenString string, secret string) (uint64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return uint64(claims["id"].(float64)), nil
	}
	return 0, fmt.Errorf("invalid token")
}
