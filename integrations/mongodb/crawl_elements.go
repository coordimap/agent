package mongodb

import (
	"context"
	"fmt"
	"sort"

	databasemodels "dev.azure.com/bloopi/bloopi/_git/shared_models.git/database_models"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (mongoCrawler *mongoCrawler) getMongodbDatabase(dbName string) databasemodels.Database {
	return databasemodels.Database{
		Name:    dbName,
		Host:    mongoCrawler.Host,
		Schemas: []string{},
	}
}

func (mongoCrawler *mongoCrawler) getMongodbDatabaseCollection(dbHandle *mongo.Database, collectionName string) (databasemodels.Table, error) {
	collectionHandle := dbHandle.Collection(collectionName)

	// get the collection indexes
	collectionIndexesNames, errListCollectionIndexesNames := mongoCrawler.listCollectionIndexesNames(collectionHandle)
	if errListCollectionIndexesNames != nil {
		// log here and do nothing else
		log.Error().Msgf("Could not retrieve index names for the collection %s. Error was: %s", collectionName, errListCollectionIndexesNames.Error())
	}

	// get collection columns
	collectionColumns, errCollectionColumns := mongoCrawler.getCollectionColumns(collectionHandle)
	if errCollectionColumns != nil {
		log.Error().Msgf("Could not retrieve columns for the collection %s. Error was: %s", collectionName, errCollectionColumns.Error())
	}

	// sort by column names
	sort.Slice(collectionColumns, func(i, j int) bool {
		return collectionColumns[i].Name < collectionColumns[j].Name
	})

	return databasemodels.Table{
		Name:        fmt.Sprintf("%s.%s", dbHandle.Name(), collectionName),
		Columns:     collectionColumns,
		Indexes:     collectionIndexesNames,
		Constraints: []databasemodels.Constraint{},
		Schema:      dbHandle.Name(),
	}, nil
}

func (mongoCrawler) listCollectionIndexesNames(collectionHandle *mongo.Collection) ([]string, error) {
	foundIndexes := []string{}
	indexesCursor, err := collectionHandle.Indexes().List(context.Background())
	if err != nil {
		return foundIndexes, err
	}

	var result []bson.M
	if err := indexesCursor.All(context.TODO(), &result); err != nil {
		log.Error().Msgf("Could not load indexes in the result to get the index names for the collection %s. Error was: %s", collectionHandle.Name(), err.Error())
	}

	for _, value := range result {
		for k, v := range value {
			if k == "name" {
				foundIndexes = append(foundIndexes, fmt.Sprintf("%s.%v", collectionHandle.Name(), v))
			}
		}
	}

	return foundIndexes, nil
}

func (mongoCrawler) listCollectionIndexes(collectionHandle *mongo.Collection) ([]databasemodels.Index, error) {
	foundIndexes := []databasemodels.Index{}
	indexesCursor, err := collectionHandle.Indexes().List(context.Background())
	if err != nil {
		return foundIndexes, err
	}

	var result []bson.M
	if err = indexesCursor.All(context.TODO(), &result); err != nil {
		log.Error().Msgf("Could not load indexes in the result to get the index details for the collection %s. Error was: %s", collectionHandle.Name(), err.Error())
	}

	for _, value := range result {
		indexName := ""
		indexCollection := ""
		indexColumns := []databasemodels.Column{}

		for k, v := range value {
			switch k {
			case "name":
				indexName = fmt.Sprintf("%v", v)

			case "ns":
				indexCollection = fmt.Sprintf("%v", v)

			case "key":
				for key := range v.(bson.M) {
					indexColumns = append(indexColumns, databasemodels.Column{
						Name:     key,
						Type:     "",
						Position: -1, // not making use of it for the time being
					})
				}
			}
		}

		foundIndexes = append(foundIndexes, databasemodels.Index{
			Name:    fmt.Sprintf("%s.%s", indexCollection, indexName),
			Columns: indexColumns,
			Table:   fmt.Sprintf("%s.%s", collectionHandle.Database().Name(), indexCollection),
			Schema:  "",
		})
	}

	return foundIndexes, nil
}

func (mongoCrawler *mongoCrawler) getCollectionColumns(collection *mongo.Collection) ([]databasemodels.Column, error) {
	allFoundColumns := []databasemodels.Column{}
	pipeline := []bson.D{{{Key: "$sample", Value: bson.D{{Key: "size", Value: 64}}}}}
	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return allFoundColumns, err
	}

	for cursor.Next(context.Background()) {
		var result bson.D
		if err := cursor.Decode(&result); err != nil {
			return allFoundColumns, err
		}

		for key, value := range result.Map() {
			var valueType string

			switch value.(type) {
			case string:
				valueType = "string"
			case int64:
				valueType = "int64"
			case primitive.D:
				valueType = "document"
			case primitive.DateTime:
				valueType = "datetime"
			case primitive.A:
				valueType = "array"
			case primitive.ObjectID:
				// TODO: try to infer references to other tables

				// TODO: create primary key constraint on the column _id

				valueType = "objectId"
			default:
				valueType = fmt.Sprintf("%T", value)
			}

			// check if column was already inserted
			columnExists := false
			for _, col := range allFoundColumns {
				if col.Name == key {
					columnExists = true
					break
				}
			}

			if !columnExists {
				allFoundColumns = append(allFoundColumns, databasemodels.Column{
					Name:     key,
					Type:     valueType,
					Position: -1,
				})
			}
		}
	}

	return allFoundColumns, nil
}
