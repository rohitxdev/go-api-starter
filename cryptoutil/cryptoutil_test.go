package cryptoutil_test

import (
	"testing"
	"time"

	"github.com/rohitxdev/go-api-starter/cryptoutil"
	"github.com/stretchr/testify/assert"
)

func TestCryptoUtil(t *testing.T) {
	t.Run("AES encryption/decryption", func(t *testing.T) {
		key := []byte("secretkey")
		plainText := []byte("Lorem ipsum dolor sit amet, consectetur adipisicing elit. Iusto itaque error, voluptates molestiae at consequuntur minima, doloremque consequatur dolores ipsam voluptatem quaerat aliquid, adipisci rem est quia nobis ducimus neque distinctio debitis. Quo exercitationem earum, possimus velit non ullam tempora, architecto maxime rerum accusantium aliquam. Fugit laborum omnis non distinctio.")

		encryptedData, err := cryptoutil.EncryptAES(plainText, key)
		assert.Nil(t, err)

		decryptedData, err := cryptoutil.DecryptAES(encryptedData, key)
		assert.Nil(t, err)

		assert.Equal(t, plainText, decryptedData)
	})

	t.Run("Argon2 password hashing", func(t *testing.T) {
		password := "password"
		hash, err := cryptoutil.HashPassword(password)
		assert.Nil(t, err)
		assert.NotEmpty(t, hash)
		assert.True(t, cryptoutil.VerifyPassword(password, hash))
	})

	t.Run("JWT generation/verification", func(t *testing.T) {
		secret := "secret"
		token, err := cryptoutil.GenerateJWT(1, time.Second*10, secret)
		assert.Nil(t, err)
		assert.NotEmpty(t, token)

		claims, err := cryptoutil.VerifyJWT(token, secret)
		assert.Nil(t, err)
		assert.NotNil(t, claims)
	})
}
