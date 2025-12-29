package services

import (
	"fmt"
	"time"

	"pulsy/internal/firebase"

	"cloud.google.com/go/firestore"
)

func CreateDoc(collection string, document string, data map[string]interface{}) error {

	client := firebase.FirestoreClient

	ctx, cancel := firebase.GetNewContextWithTimeout(3 * time.Second)
	defer cancel()

	_, err := client.Collection(collection).Doc(document).Set(ctx, data)

	if err != nil {
		return fmt.Errorf("failed insert data: %v", err)
	}

	return nil
}

func ReadDoc(collection string, document string) (map[string]interface{}, error) {
	client := firebase.FirestoreClient

	ctx, cancel := firebase.GetNewContextWithTimeout(3 * time.Second)
	defer cancel()

	doc, err := client.Collection(collection).Doc(document).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting document: %v", err)
	}

	return doc.Data(), nil
}

func UpdateDoc(collection string, document string, data map[string]interface{}) error {
	client := firebase.FirestoreClient

	ctx, cancel := firebase.GetNewContextWithTimeout(3 * time.Second)
	defer cancel()

	_, err := client.Collection(collection).Doc(document).Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("error updating document: %v", err)
	}

	return nil
}

func DeleteDoc(collection string, document string) error {
	client := firebase.FirestoreClient

	ctx, cancel := firebase.GetNewContextWithTimeout(3 * time.Second)
	defer cancel()

	_, err := client.Collection(collection).Doc(document).Delete(ctx)
	if err != nil {
		return fmt.Errorf("error deleting document: %v", err)
	}

	return nil
}

func ReadMultipleDocs(collection string, conditions []firebase.QueryCondition) ([]map[string]interface{}, error) {
	client := firebase.FirestoreClient

	ctx, cancel := firebase.GetNewContextWithTimeout(3 * time.Second)
	defer cancel()

	query := client.Collection(collection).Query

	// Aplicar condiciones a la consulta
	for _, condition := range conditions {
		query = query.Where(condition.Field, condition.Operator, condition.Value)
	}

	// Ejecutar la consulta
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("error querying documents: %v", err)
	}

	// Parsear resultados
	var results []map[string]interface{}
	for _, doc := range docs {
		// Obtener los datos del documento como map[string]interface{}
		data := doc.Data()

		// Agregar el ID del documento al mapa
		data["docRefID"] = doc.Ref.ID

		// Agregar al slice de resultados
		results = append(results, data)
	}

	return results, nil
}
