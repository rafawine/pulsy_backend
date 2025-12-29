package routes

import (
	"os"
	"pulsy/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	// Set the Gin to release mode
	gin.SetMode(os.Getenv("GIN_MODE"))

	router := gin.Default()

	// Health Check
	router.GET("/health", handlers.HealthCheckHandler)

	// File routes V3.0.0
	fileRouter := router.Group("/files")

	// Upload file to storage
	fileRouter.POST("/", handlers.UploadFileHandler)

	// Get list of metadata of all files
	fileRouter.GET("/", handlers.GetListFileMetadataHandler)

	// Get metadata of specific file
	fileRouter.GET("/:uuid/metadata", handlers.GetFileMetadataHandler)

	// Download file from storage
	fileRouter.GET("/:uuid", handlers.DownloadFileHandler)

	// Update file to storage
	fileRouter.PUT("/:uuid", handlers.UpdateFileHandler)

	// Delete file from storage
	fileRouter.DELETE("/:uuid", handlers.DeleteFileHandler)

	return router
}
