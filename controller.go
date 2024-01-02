package main

import (
	"os"

	"github.com/gofiber/fiber"
)

func httpController() {
	app := fiber.New()

	app.Get("/_health", func(c *fiber.Ctx) {
		c.SendString("☕")
	})

	app.Put("/exit", func(c *fiber.Ctx) {
		os.Exit(0)
	})

	app.Listen(":3000")
}
