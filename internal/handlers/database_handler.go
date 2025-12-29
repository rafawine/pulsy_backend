package handlers

import (
	"net/http"
	"pulsy/internal/firebase"
	"pulsy/internal/responses"
	"pulsy/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetListFileMetadataHandler(c *gin.Context) {
	ownerUUID := c.Request.Header.Get("x-consumer-custom-id")

	// Criterios de búsqueda
	conditions := []firebase.QueryCondition{
		{Field: "ownerUUID", Operator: "==", Value: ownerUUID},
	}

	// Obtener metadatos del los archivos
	fileMetadata, err := services.ReadMultipleDocs("files", conditions)
	if err != nil {
		mesage := "failed to get file metadata"
		errData := map[string]interface{}{
			"cause": err.Error(),
		}

		c.IndentedJSON(http.StatusConflict, responses.Error(mesage, errData))
		return
	}

	// Limpiar metadatos
	fileMetadata = cleanFilesMetadata(fileMetadata)

	// Responder con éxito
	message := "file metadata obtained successfully"
	successData := fileMetadata

	c.IndentedJSON(http.StatusOK, responses.Success(message, successData))
}

func GetFileMetadataHandler(c *gin.Context) {
	ownerUUID := c.Request.Header.Get("x-consumer-custom-id")
	fileUUID := c.Param("uuid")

	//var conditions []firebase.QueryCondition
	var fileMetadata map[string]interface{}

	// Casos para los criterios de búsqueda
	if _, err := uuid.Parse(fileUUID); err == nil {
		// Obtener metadatos del archivo
		fileMetadata, err = services.ReadDoc("files", fileUUID)
		if err != nil {
			mesage := "failed to get file metadata"
			errData := map[string]interface{}{
				"uuid":  fileUUID,
				"cause": err.Error(),
			}

			c.IndentedJSON(http.StatusConflict, responses.Error(mesage, errData))
			return
		}

		if !(fileMetadata["ownerUUID"] == ownerUUID) {
			//return nil, fmt.Errorf("file not found, verify the uuid")
			mesage := "failed to get file metadata"
			errData := map[string]interface{}{
				"uuid": fileUUID,
			}

			c.IndentedJSON(http.StatusConflict, responses.Error(mesage, errData))
			return
		}
	} else {
		// Handle invalid input
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file identifier provided.",
		})
		return
	}

	// Limpiar metadatos
	delete(fileMetadata, "ownerUUID")

	// Responder con éxito
	message := "file metadata obtained successfully"
	successData := fileMetadata

	c.IndentedJSON(http.StatusOK, responses.Success(message, successData))
}

func cleanFilesMetadata(fileMetadata []map[string]interface{}) []map[string]interface{} {
	for _, itemfileMetadata := range fileMetadata {
		itemfileMetadata["uuid"] = itemfileMetadata["docRefID"].(string)

		delete(itemfileMetadata, "ownerUUID")
		delete(itemfileMetadata, "docRefID")
	}

	return fileMetadata
}
