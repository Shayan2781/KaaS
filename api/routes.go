package api

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
)

const port = "8080"

func Routes(server *echo.Echo) {
	server.POST("/deploy-unmanaged", DeployUnmanagedObjects)
	server.POST("/deploy-managed", DeployManagedObjects)
	if err := server.Start(fmt.Sprintf("localhost:%s", port)); err != nil {
		log.Fatalf("Server failed to listen: %v", err)
	}
}
