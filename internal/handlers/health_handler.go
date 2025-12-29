package handlers

import (
	"net/http"

	"pulsy/internal/responses"

	"github.com/gin-gonic/gin"
)

func HealthCheckHandler(c *gin.Context) {
	message := "pulsy API is running"
	data := map[string]interface{}{
		"version": "3.0.0",
	}

	c.IndentedJSON(http.StatusAccepted, responses.Success(message, data))
}
