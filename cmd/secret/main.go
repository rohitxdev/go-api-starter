package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	prefix := ""
	charLen := 64
	encoding := "base64url"

	fmt.Printf("Prefix (e.g. whsec, sk_live)[%s]: ", prefix)
	if input, err := reader.ReadString('\n'); err == nil {
		input = strings.TrimSuffix(strings.TrimSpace(input), "_")
		if input != "" {
			prefix = input + "_"
		}
	}

	fmt.Printf("Length (characters) [%d]: ", charLen)
	if input, err := reader.ReadString('\n'); err == nil {
		input = strings.TrimSpace(input)
		if input != "" {
			v, err := strconv.Atoi(input)
			if err != nil || v <= 0 {
				log.Fatal("invalid length")
			}
			charLen = v
		}
	}

	fmt.Printf("Encoding (hex/base64url) [%s]: ", encoding)
	if input, err := reader.ReadString('\n'); err == nil {
		input = strings.TrimSpace(input)
		if input != "" {
			encoding = input
		}
	}

	var byteLen int
	switch encoding {
	case "hex":
		byteLen = int(math.Ceil(float64(charLen) / 2))
	case "base64url":
		byteLen = int(math.Ceil(float64(charLen) * 3 / 4))
	default:
		log.Fatal("unsupported encoding")
	}

	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		log.Fatal(err)
	}

	var secret string
	switch encoding {
	case "hex":
		secret = hex.EncodeToString(b)
	case "base64url":
		secret = base64.RawURLEncoding.EncodeToString(b)
	}

	secret = prefix + secret
	secret = secret[:charLen]

	fmt.Printf("\n🔑 Generated secret:\n\t%v\n", secret)
}
