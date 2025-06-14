package main

import (
	"github.com/romanpitatelev/hezzl-goods/internal/app"
	"github.com/romanpitatelev/hezzl-goods/internal/configs"
)

func main() {
	cfg := configs.New()

	if err := app.Run(cfg); err != nil {
		panic(err)
	}
}
