package inventory

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	collection *mongo.Collection
	now        func() time.Time
}

func NewMongoRepository(database *mongo.Database) *MongoRepository {
	return &MongoRepository{collection: database.Collection("inventory"), now: time.Now}
}

func (repository *MongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := repository.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "product_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("unique_product_inventory"),
	})
	return err
}

func (repository *MongoRepository) Adjust(ctx context.Context, productID bson.ObjectID, delta int64) (Inventory, bool, error) {
	filter := bson.D{{Key: "product_id", Value: productID}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{{Key: "quantity", Value: delta}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: repository.now().UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "_id", Value: bson.NewObjectID()}, {Key: "product_id", Value: productID}}},
	}
	upsert := delta > 0
	if delta < 0 {
		filter = append(filter, bson.E{Key: "quantity", Value: bson.D{{Key: "$gte", Value: -delta}}})
	}

	var stock Inventory
	err := repository.collection.FindOneAndUpdate(
		ctx,
		filter,
		update,
		options.FindOneAndUpdate().SetUpsert(upsert).SetReturnDocument(options.After),
	).Decode(&stock)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Inventory{}, false, nil
	}
	if mongo.IsDuplicateKeyError(err) && upsert {
		return repository.adjustExisting(ctx, productID, delta)
	}
	if err != nil {
		return Inventory{}, false, err
	}
	return stock, true, nil
}

func (repository *MongoRepository) adjustExisting(ctx context.Context, productID bson.ObjectID, delta int64) (Inventory, bool, error) {
	var stock Inventory
	err := repository.collection.FindOneAndUpdate(
		ctx,
		bson.D{{Key: "product_id", Value: productID}},
		bson.D{
			{Key: "$inc", Value: bson.D{{Key: "quantity", Value: delta}}},
			{Key: "$set", Value: bson.D{{Key: "updated_at", Value: repository.now().UTC()}}},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&stock)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Inventory{}, false, nil
	}
	return stock, err == nil, err
}

func (repository *MongoRepository) Get(ctx context.Context, productID bson.ObjectID) (Inventory, error) {
	var stock Inventory
	err := repository.collection.FindOne(ctx, bson.D{{Key: "product_id", Value: productID}}).Decode(&stock)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Inventory{}, ErrNotFound
	}
	return stock, err
}
