package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))

	e.GET("/ping", func(c echo.Context) error {
		fmt.Println("pinged")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "pong",
		})
	})

	e.GET("/health", func(ctx echo.Context) error {
		fmt.Println("health checked")
		return ctx.JSON(http.StatusOK, map[string]interface{}{
			"status": "ok!",
		})
	})

	e.Start(":4883")
}
