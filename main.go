package main

import (
	"KaaS/api"
	"KaaS/confgs"
	"github.com/labstack/echo/v4"
)

func main() {
	confgs.CreateClient()
	server := echo.New()
	api.Routes(server)
}
