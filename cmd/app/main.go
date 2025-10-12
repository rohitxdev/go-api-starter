package main

import (
	"fmt"

	"github.com/rohitxdev/go-api/application"
)

func main() {
	if err := application.Run(); err != nil {
		panic(fmt.Errorf("failed to run application: %w", err))
	}
}
