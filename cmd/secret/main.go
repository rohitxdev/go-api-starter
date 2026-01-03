package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/rohitxdev/go-api/util"
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

	secret, err := util.GenerateSecret(prefix, charLen, encoding)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to generate secret: %w", err))
	}

	fmt.Printf("\n🔑 Generated secret:\n\t%v\n", secret)
}
