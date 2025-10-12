package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
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
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to create nonce: %w", err)
	}
	//Append cipher to nonce and return nonce + cipher
	return gcm.Seal(nonce, nonce, data, nil), nil
}

// Decrypts data using AES algorithm. The key should be same key that was used to encrypt the data.
func DecryptAES(encryptedData []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(padKey(key))
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()

	//Get nonce from encrypted data
	nonce, cipher := encryptedData[:nonceSize], encryptedData[nonceSize:]
	data, err := gcm.Open(nil, nonce, cipher, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
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

const (
	hashTime    = 4         // number of iterations
	hashMemory  = 64 * 1024 // memory in KiB
	hashThreads = 4         // parallelism
	hashKeyLen  = 32        // length of the generated key
	saltLen     = 16        // length of the random salt
)

func hashArgon2(data, salt []byte) []byte {
	return argon2.IDKey(data, salt, hashTime, hashMemory, hashThreads, hashKeyLen)
}

// GenerateSecureHash generates a salted Argon2 hash of the given data.
func GenerateSecureHash(data []byte) ([]byte, error) {
	salt, err := generateSalt(saltLen)
	if err != nil {
		return nil, err
	}
	hash := hashArgon2(data, salt)
	return append(salt, hash...), nil
}

// VerifySecureHash verifies the given data against a salted hash produced by GenerateSecureHash.
func VerifySecureHash(data []byte, saltedHash []byte) bool {
	if len(saltedHash) < saltLen {
		return false
	}
	salt := saltedHash[:saltLen]
	hash := saltedHash[saltLen:]
	computed := hashArgon2(data, salt)
	return subtle.ConstantTimeCompare(hash, computed) == 1
}

func GenerateJWT[T any](data T, expiresIn time.Duration, secret string) (string, error) {
	t := time.Now()
	claims := jwt.MapClaims{
		"data": data,
		"iat":  t.Unix(),
		"nbf":  t.Unix(),
		"exp":  t.Add(expiresIn).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

// The type of claims may not be the same as the type given during the token creation. For example, non-float numbers get converted to float when parsed due to how JWT processes data. Be cautious and don't put non-primitive types in claims.
func verifyJWTUnsafe[T any](tokenStr string, secret string) (T, error) {
	var data T
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return data, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if err = claims.Valid(); err != nil {
			return data, err
		}
		if data, ok = claims["data"].(T); ok {
			return data, nil
		}
		return data, fmt.Errorf("expected data to have type %T, but got %T", data, claims["data"])
	}
	return data, errors.New("invalid token")
}

func convertToType[T any](data any) (T, bool) {
	var zero T
	targetType := reflect.TypeOf(zero)

	// Special handling for maps
	if targetType.Kind() == reflect.Map {
		dataValue := reflect.ValueOf(data)
		if dataValue.Kind() == reflect.Map {
			// Create a new map of the target type
			newMap := reflect.MakeMap(targetType)

			// Iterate through source map and convert each key-value pair
			iter := dataValue.MapRange()
			for iter.Next() {
				k := iter.Key()
				v := iter.Value()

				// Ensure correct type conversion for keys and values
				convertedK := k.Interface()
				convertedV := v.Interface()

				newMap.SetMapIndex(reflect.ValueOf(convertedK), reflect.ValueOf(convertedV))
			}

			return newMap.Interface().(T), true
		}
	}

	// General type conversion
	if reflect.TypeOf(data).ConvertibleTo(targetType) {
		converted := reflect.ValueOf(data).Convert(targetType).Interface()
		return converted.(T), true
	}

	return zero, false
}

// Works only for primitive types and maps.
func VerifyJWT[T any](tokenStr string, secret string) (T, error) {
	var zero T
	anyData, err := verifyJWTUnsafe[any](tokenStr, secret)
	if err != nil {
		return zero, err
	}
	if data, ok := convertToType[T](anyData); ok {
		return data, nil
	}
	return zero, fmt.Errorf("expected data to have type %T, but got %T", zero, anyData)
}

// Exclude similar looking characters like 0, O, I, 1, l
const alphaNumCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func GenerateAlphaNumCode(size int) (string, error) {
	if size < 0 {
		return "", errors.New("size must be non-negative")
	}

	charsetSize := big.NewInt(int64(len(alphaNumCharset)))
	var code strings.Builder
	code.Grow(size)

	for range size {
		n, err := rand.Int(rand.Reader, charsetSize)
		if err != nil {
			return "", fmt.Errorf("failed to create random integer: %w", err)
		}
		code.WriteByte(alphaNumCharset[n.Int64()])
	}
	return code.String(), nil
}
