package handlers

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pulsy/internal/firebase"
	"pulsy/internal/responses"
	"pulsy/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func UploadFileHandler(c *gin.Context) {
	uuidOwner := c.Request.Header.Get("x-consumer-custom-id")
	file, err := c.FormFile("file")
	if err != nil {
		message := "file not found"
		errData := map[string]interface{}{
			"field": "file",
			"cause": "file is required",
		}

		c.IndentedJSON(http.StatusBadRequest, responses.Error(message, errData))
		return
	}

	// Generar un UUID único y construir el nombre del archivo en el bucket
	fileUUID := uuid.New().String()
	fileFullName := file.Filename
	fileExtension := filepath.Ext(fileFullName)
	fileName := strings.Replace(fileFullName, fileExtension, "", -1)
	fileSize := file.Size
	bucketFileName := fileUUID + fileExtension

	metadata := map[string]interface{}{
		"uuid":         fileUUID[:8],
		"fullName":     fileFullName,
		"name":         fileName,
		"extension":    fileExtension,
		"size":         fileSize,
		"creationDate": time.Now(),
		"uuidOwner":    uuidOwner,
	}

	// Guardar archivo en Storage
	err = services.UploadFile(os.Getenv("BUCKET"), bucketFileName, file)
	if err != nil {
		message := "failed to upload file"
		errData := map[string]interface{}{
			"uuid":          fileUUID,
			"fileName":      fileName,
			"fileExtension": fileExtension,
			"fileSize":      fileSize,
			"cause":         err.Error(),
		}

		c.IndentedJSON(http.StatusConflict, responses.Error(message, errData))
		return
	}

	// Guardar metadatos en Firestore
	err = services.CreateDoc("files", fileUUID, metadata)
	if err != nil {
		// Eliminar el archivo del bucket si hubo un error
		bucketErr := services.DeleteFile(os.Getenv("BUCKET"), bucketFileName)

		var message string

		if bucketErr != nil {
			message = "failed to delete file"
			err = bucketErr
		} else {
			message = "failed to insert metadata"
		}

		errData := map[string]interface{}{
			"uuid":          fileUUID,
			"fileName":      fileName,
			"fileExtension": fileExtension,
			"fileSize":      fileSize,
			"cause":         err.Error(),
		}

		c.IndentedJSON(http.StatusConflict, responses.Error(message, errData))
		return
	}

	// Responder con éxito
	message := "files uploaded successfully"
	successData := map[string]interface{}{
		"uuid":          fileUUID,
		"fileName":      fileName,
		"fileExtension": fileExtension,
		"fileFullName":  fileFullName,
		"fileSize":      fileSize,
	}

	c.IndentedJSON(http.StatusAccepted, responses.Success(message, successData))
}

func DownloadFileHandler(c *gin.Context) {
	uuidOwner := c.Request.Header.Get("x-consumer-custom-id")
	fileUUID := c.Param("uuid")

	metadata, err := getMetadata(fileUUID, uuidOwner)
	if err != nil {
		message := "failed to get metadata"
		errData := map[string]interface{}{
			"uuid":  fileUUID,
			"cause": err.Error(),
		}

		c.IndentedJSON(http.StatusNotFound, responses.Error(message, errData))
		return
	}

	if len(fileUUID) == 8 {
		fileUUID = metadata["idDoc"].(string)
	}

	fileFullName := metadata["fullName"].(string)
	fileName := metadata["name"].(string)
	fileExtension := metadata["extension"].(string)
	fileSize := metadata["size"].(int64)
	bucketFileName := fileUUID + fileExtension

	// Crear archivo temporal seguro
	tempFile, err := os.CreateTemp("", fileFullName)
	if err != nil {
		message := "failed to create temporary file"
		errData := map[string]interface{}{
			"uuid":          fileUUID,
			"fileName":      fileName,
			"fileExtension": fileExtension,
			"fileSize":      fileSize,
			"cause":         err.Error(),
		}

		c.IndentedJSON(http.StatusInternalServerError, responses.Error(message, errData))
		return
	}
	defer os.Remove(tempFile.Name()) // Se asegura que el archivo sea eliminado al finalizar

	// Descargar el archivo al sistema local
	err = services.DownloadFile(os.Getenv("BUCKET"), bucketFileName, tempFile.Name())
	if err != nil {
		message := "failed to download file"
		errData := map[string]interface{}{
			"uuid":          fileUUID,
			"fileName":      fileName,
			"fileExtension": fileExtension,
			"fileSize":      fileSize,
			"cause":         err.Error(),
		}

		c.IndentedJSON(http.StatusInternalServerError, responses.Error(message, errData))
		return
	}

	// Usar mime.TypeByExtension para obtener el tipo MIME
	contentType := mime.TypeByExtension(fileExtension)

	if contentType == "" {
		contentType = "application/octet-stream" // Tipo predeterminado si no se encuentra
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+fileFullName)
	c.Header("Content-Length", fmt.Sprintf("%d", fileSize))

	// Enviar el archivo como respuesta
	c.File(tempFile.Name())
}

func UpdateFileHandler(c *gin.Context) {
	uuidOwner := c.Request.Header.Get("x-consumer-custom-id")
	fileUUID := c.Param("uuid")
	file, err := c.FormFile("file")
	if err != nil {
		message := "file not found"
		errData := map[string]interface{}{
			"field": "file",
			"cause": "file is required",
		}

		c.IndentedJSON(http.StatusBadRequest, responses.Error(message, errData))
		return
	}

	metadata, err := getMetadata(fileUUID, uuidOwner)
	if err != nil {
		message := "failed to get metadata"
		errData := map[string]interface{}{
			"uuid":  fileUUID,
			"cause": err.Error(),
		}

		c.IndentedJSON(http.StatusNotFound, responses.Error(message, errData))
		return
	}

	if len(fileUUID) == 8 {
		fileUUID = metadata["idDoc"].(string)
	}

	fileExtension := metadata["extension"].(string)

	// Construir el nombre del archivo en el bucket
	newFileFullName := file.Filename
	newFileExtension := filepath.Ext(newFileFullName)
	newFileName := strings.Replace(newFileFullName, newFileExtension, "", -1)
	newFileSize := file.Size
	bucketFileName := fileUUID + newFileExtension

	// Verificar si se trata del mismo archivo
	if !(newFileExtension == fileExtension) {
		message := "failed to upload file"
		errData := map[string]interface{}{
			"uuid":          fileUUID,
			"fileName":      newFileName,
			"fileExtension": newFileExtension,
			"fileSize":      newFileSize,
			"cause":         "different type of file",
		}

		c.IndentedJSON(http.StatusConflict, responses.Error(message, errData))
		return
	}

	newMetadata := map[string]interface{}{
		"fullName": newFileFullName,
		"name":     newFileName,
		"size":     newFileSize,
	}

	// Actualizar archivo en Storage
	err = services.UploadFile(os.Getenv("BUCKET"), bucketFileName, file)
	if err != nil {
		message := "failed to upload file"
		errData := map[string]interface{}{
			"uuid":          fileUUID,
			"fileName":      newFileName,
			"fileExtension": newFileExtension,
			"fileSize":      newFileSize,
			"cause":         err.Error(),
		}

		c.IndentedJSON(http.StatusConflict, responses.Error(message, errData))
		return
	}

	// Actualizar metadatos en Firestore
	err = services.UpdateDoc("files", fileUUID, newMetadata)
	if err != nil {
		message := "failed to update metadata"
		errData := map[string]interface{}{
			"uuid":          fileUUID,
			"fileName":      newFileName,
			"fileExtension": newFileExtension,
			"fileSize":      newFileSize,
			"cause":         err.Error(),
		}

		c.IndentedJSON(http.StatusConflict, responses.Error(message, errData))
		return
	}

	// Responder con éxito
	message := "files uploaded successfully"
	successData := map[string]interface{}{
		"uuid":          fileUUID,
		"fileName":      newFileName,
		"fileExtension": newFileExtension,
		"fileFullName":  newFileFullName,
		"fileSize":      newFileSize,
	}

	c.IndentedJSON(http.StatusAccepted, responses.Success(message, successData))
}

func DeleteFileHandler(c *gin.Context) {
	uuidOwner := c.Request.Header.Get("x-consumer-custom-id")
	fileUUID := c.Param("uuid")

	metadata, err := getMetadata(fileUUID, uuidOwner)
	if err != nil {
		message := "failed to get metadata"
		errData := map[string]interface{}{
			"uuid":  fileUUID,
			"cause": err.Error(),
		}

		c.IndentedJSON(http.StatusNotFound, responses.Error(message, errData))
		return
	}

	if len(fileUUID) == 8 {
		fileUUID = metadata["idDoc"].(string)
	}

	fileFullName := metadata["fullName"].(string)
	fileName := metadata["name"].(string)
	fileExtension := metadata["extension"].(string)
	fileSize := metadata["size"].(int64)
	bucketFileName := fileUUID + fileExtension

	// Eliminar el archivo del bucket
	err = services.DeleteFile(os.Getenv("BUCKET"), bucketFileName)
	if err != nil {
		message := "failed to delete file"
		errData := map[string]interface{}{
			"uuid":          fileUUID,
			"fileName":      fileName,
			"fileExtension": fileExtension,
			"fileSize":      fileSize,
			"cause":         err.Error(),
		}

		c.IndentedJSON(http.StatusInternalServerError, responses.Error(message, errData))
		return
	}

	// Eliminar registro de Firestore
	err = services.DeleteDoc("files", fileUUID)
	if err != nil {
		message := "failed to delete metadata"
		errData := map[string]interface{}{
			"uuid":          fileUUID,
			"fileName":      fileName,
			"fileExtension": fileExtension,
			"fileSize":      fileSize,
			"cause":         err.Error(),
		}

		c.IndentedJSON(http.StatusInternalServerError, responses.Error(message, errData))
		return
	}

	// Responder con éxito
	message := "files delete successfully"
	successData := map[string]interface{}{
		"uuid":          fileUUID,
		"fileName":      fileName,
		"fileExtension": fileExtension,
		"fileFullName":  fileFullName,
		"fileSize":      fileSize,
	}

	c.IndentedJSON(http.StatusAccepted, responses.Success(message, successData))
}

func getMetadata(uuidFile string, uuidOwner string) (map[string]interface{}, error) {
	// Verificar si el UUID es de 8 caracteres
	if len(uuidFile) == 8 {
		// Criterios de búsqueda
		conditions := []firebase.QueryCondition{
			{Field: "uuid", Operator: "==", Value: uuidFile},
			{Field: "uuidOwner", Operator: "==", Value: uuidOwner},
		}

		// Obtener metadatos del archivo
		metadata, err := services.ReadMultipleDocs("files", conditions)
		if err != nil {
			return nil, err
		}

		if len(metadata) == 0 {
			return nil, fmt.Errorf("file not found, verify the uuid")
		}

		if len(metadata) > 1 {
			return nil, fmt.Errorf("multiple files found, insert a file full uuid")
		}

		return metadata[0], nil
	} else {
		_, err := uuid.Parse(uuidFile)
		if err != nil {
			return nil, err
		}

		// Obtener metadatos del archivo
		metadata, err := services.ReadDoc("files", uuidFile)
		if err != nil {
			return nil, err
		}

		if !(metadata["uuidOwner"] == uuidOwner) {
			return nil, fmt.Errorf("file not found, verify the uuid")
		}

		return metadata, nil
	}
}
