package main

import (
	"github.com/romanpitatelev/denet/internal/app"
	"github.com/romanpitatelev/denet/internal/configs"
)

func main() {
	cfg := configs.New()

	if err := app.Run(cfg); err != nil {
		panic(err)
	}
}
