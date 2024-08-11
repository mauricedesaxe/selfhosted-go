package cmd

import (
	"fmt"
	"go-on-rails/auth"
	"go-on-rails/common"
	"go-on-rails/marketing"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func Run() {
	// prepare app
	log.Println("Starting server on port", common.Env.PORT)
	app := fiber.New()
	app.Use(logger.New())

	// add routes
	app.Static("/", "./public")
	marketing.AddRoutes(app)
	auth.AddRoutes(app)

	// start server
	err := app.Listen(fmt.Sprintf(":%s", common.Env.PORT))
	if err != nil {
		log.Println("Error starting server")
		log.Println(err)
	}
}
