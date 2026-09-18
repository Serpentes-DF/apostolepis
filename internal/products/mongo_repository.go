package products

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	collection *mongo.Collection
}

func NewMongoRepository(database *mongo.Database) *MongoRepository {
	return &MongoRepository{collection: database.Collection("products")}
}

func (repository *MongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := repository.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "sku", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("unique_sku"),
	})
	return err
}

func (repository *MongoRepository) Create(ctx context.Context, product Product) (Product, error) {
	if product.ID.IsZero() {
		product.ID = bson.NewObjectID()
	}
	if _, err := repository.collection.InsertOne(ctx, product); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return Product{}, ErrDuplicate
		}
		return Product{}, err
	}
	return product, nil
}

func (repository *MongoRepository) Get(ctx context.Context, id bson.ObjectID) (Product, error) {
	var product Product
	err := repository.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&product)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Product{}, ErrNotFound
	}
	return product, err
}

func (repository *MongoRepository) List(ctx context.Context) ([]Product, error) {
	cursor, err := repository.collection.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{
		{Key: "created_at", Value: 1}, {Key: "_id", Value: 1},
	}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	products := make([]Product, 0)
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}
	return products, nil
}

func (repository *MongoRepository) Update(ctx context.Context, product Product) (Product, error) {
	result, err := repository.collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: product.ID}}, product)
	if mongo.IsDuplicateKeyError(err) {
		return Product{}, ErrDuplicate
	}
	if err != nil {
		return Product{}, err
	}
	if result.MatchedCount == 0 {
		return Product{}, ErrNotFound
	}
	return product, nil
}

func (repository *MongoRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	result, err := repository.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (repository *MongoRepository) Exists(ctx context.Context, id bson.ObjectID) (bool, error) {
	err := repository.collection.FindOne(
		ctx,
		bson.D{{Key: "_id", Value: id}},
		options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 1}}),
	).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}
