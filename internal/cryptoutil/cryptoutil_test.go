package cryptoutil_test

import (
	"testing"

	"github.com/rohitxdev/go-api-starter/internal/cryptoutil"
	"github.com/stretchr/testify/assert"
)

func TestAES(t *testing.T) {
	key := []byte("1234567812345678")
	plainText := []byte("Lorem ipsum dolor sit amet, consectetur adipisicing elit. Iusto itaque error, voluptates molestiae at consequuntur minima, doloremque consequatur dolores ipsam voluptatem quaerat aliquid, adipisci rem est quia nobis ducimus neque distinctio debitis. Quo exercitationem earum, possimus velit non ullam tempora, architecto maxime rerum accusantium aliquam. Fugit laborum omnis non distinctio.")

	encryptedData, err := cryptoutil.EncryptAES(plainText, key)
	assert.Nil(t, err)

	decryptedData, err := cryptoutil.DecryptAES(encryptedData, key)
	assert.Nil(t, err)

	assert.Equal(t, plainText, decryptedData)
}
