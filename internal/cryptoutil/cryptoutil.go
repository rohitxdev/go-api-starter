// Package cryptoutil provides utility functions for encryption and decryption.
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
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

func bufToBase62(buf []byte) string {
	var i big.Int
	i.SetBytes(buf)
	return i.Text(62)
}

func RandomString() string {
	var buf = make([]byte, 64)
	_, _ = rand.Read(buf)
	return bufToBase62(buf)
}

func Base62Hash(text string) string {
	hasher := sha256.New()
	buf := hasher.Sum([]byte(text))
	return bufToBase62(buf)
}
