package mongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"mxtransporter/config/mongodb"
	"mxtransporter/pkg/common"
	"mxtransporter/pkg/errors"
)

var (
	mongoConfig = mongodb.MongoConfig()
)

func Health(ctx context.Context, client *mongo.Client) error {
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return errors.InternalServerErrorMongoDbConnect.Wrap("Failed to ping mongodb.", err)
	}
	return nil
}

func FetchDatabase(ctx context.Context, client *mongo.Client) (*mongo.Database, error) {
	dbList, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return nil, errors.InternalServerErrorMongoDbOperate.Wrap("Failed to list databases.", err)
	}

	if !common.Contains(dbList, mongoConfig.MongoDbDatabase) {
		return nil, errors.InternalServerErrorMongoDbOperate.New("The specified mongodb database does not exist.")
	}
	db := client.Database(mongoConfig.MongoDbDatabase)
	return db, nil
}

func FetchCollection(ctx context.Context, db *mongo.Database) (*mongo.Collection, error) {
	collList, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, errors.InternalServerErrorMongoDbOperate.Wrap("Failed to list collections.", err)
	}

	if !common.Contains(collList, mongoConfig.MongoDbCollection) {
		return nil, errors.InternalServerErrorMongoDbOperate.New("The specified mongodb collection does not exist.")
	}
	cl := db.Collection(mongoConfig.MongoDbCollection)
	return cl, nil
}
