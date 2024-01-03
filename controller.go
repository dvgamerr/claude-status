package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

var apiPort string

func httpController() {
	app := fiber.New()

	app.Get("/_health", func(c *fiber.Ctx) error {
		return c.SendString("☕\n")
	})

	app.Put("/_exit", func(c *fiber.Ctx) error {
		go delayedExit()
		return c.SendString("Signal Exiting...\n")
	})

	apiPort = os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "21280"
	}

	sugar.Infof("Listen backend port: %s\n", apiPort)
	app.Listen(fmt.Sprintf(":%s", apiPort))
}

func delayedExit() {
	sugar.Infoln("Signal Exiting...")
	sugar.Sync()
	time.Sleep(200 * time.Millisecond)
	fmt.Print("\033[H\033[2J")
	os.Exit(0)
}
