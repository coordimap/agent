package mongodb

import (
	"context"
	"fmt"
	"log"

	dbModel "dev.azure.com/bloopi/bloopi/_git/shared_models.git/postgres"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (mongoCrawler *mongoCrawler) getMongodbDatabase(dbName string) dbModel.Database {
	return dbModel.Database{
		Name:    dbName,
		Host:    mongoCrawler.Host,
		Schemas: []string{},
	}
}

func (mongoCrawler *mongoCrawler) getMongodbDatabaseCollection(dbHandle *mongo.Database, collectionName string) (dbModel.Table, error) {
	collectionHandle := dbHandle.Collection(collectionName)

	// get the collection indexes
	collectionIndexesNames, errListCollectionIndexesNames := mongoCrawler.listCollectionIndexesNames(collectionHandle)
	if errListCollectionIndexesNames != nil {
		// TODO: log here and do nothing else
	}

	// TODO: sort by column names
	return dbModel.Table{
		Indexes: collectionIndexesNames,
	}, nil
}

func (mongoCrawler) listCollectionIndexesNames(collectionHandle *mongo.Collection) ([]string, error) {
	foundIndexes := []string{}
	indexesCursor, err := collectionHandle.Indexes().List(context.Background())
	if err != nil {
		return foundIndexes, err
	}

	var result []bson.M
	if err = indexesCursor.All(context.TODO(), &result); err != nil {
		log.Fatal(err)
	}

	for _, value := range result {
		for k, v := range value {
			if k == "name" {
				foundIndexes = append(foundIndexes, fmt.Sprintf("%v", v))
			}
		}
	}

	return foundIndexes, nil
}

func (mongoCrawler) listCollectionIndexes(collectionHandle *mongo.Collection) ([]dbModel.Index, error) {
	foundIndexes := []dbModel.Index{}
	indexesCursor, err := collectionHandle.Indexes().List(context.Background())
	if err != nil {
		return foundIndexes, err
	}

	var result []bson.M
	if err = indexesCursor.All(context.TODO(), &result); err != nil {
		log.Fatal(err)
	}

	for _, value := range result {
		indexName := ""
		indexCollection := ""
		indexColumns := []dbModel.Column{}

		for k, v := range value {
			switch k {
			case "name":
				indexName = fmt.Sprintf("%v", v)

			case "ns":
				indexCollection = fmt.Sprintf("%v", v)

			case "key":
				for key, _ := range v.(map[string]int) {
					indexColumns = append(indexColumns, dbModel.Column{
						Name:     key,
						Type:     "",
						Position: -1, // not making use of it for the time being
					})
				}
			}
		}

		foundIndexes = append(foundIndexes, dbModel.Index{
			Name:    fmt.Sprintf("%s.%s", indexCollection, indexName),
			Columns: indexColumns,
			Table:   indexCollection,
			Schema:  "",
		})
	}

	return foundIndexes, nil
}
