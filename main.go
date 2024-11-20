//go:generate go run github.com/swaggo/swag/cmd/swag@latest init -q -g handler/router.go
package main

import (
	"github.com/rohitxdev/go-api-starter/app"
	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	if _, err := maxprocs.Set(); err != nil {
		panic("Failed to set maxprocs: " + err.Error())
	}
	if err := app.Run(); err != nil {
		panic(err)
	}
}
