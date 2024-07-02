package main

import (
	"KaaS/api"
	"KaaS/configs"
	"github.com/labstack/echo/v4"
)

func main() {
	configs.CreateClient()
	server := echo.New()
	api.Routes(server)
}
