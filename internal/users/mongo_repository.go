package users

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
	return &MongoRepository{collection: database.Collection("users")}
}

func (repository *MongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := repository.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("unique_email"),
	})
	return err
}

func (repository *MongoRepository) Create(ctx context.Context, user User) (User, error) {
	if user.ID.IsZero() {
		user.ID = bson.NewObjectID()
	}
	if _, err := repository.collection.InsertOne(ctx, user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return User{}, ErrDuplicate
		}
		return User{}, err
	}
	return user, nil
}

func (repository *MongoRepository) Get(ctx context.Context, id bson.ObjectID) (User, error) {
	var user User
	err := repository.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (repository *MongoRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := repository.collection.FindOne(ctx, bson.D{{Key: "email", Value: email}}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (repository *MongoRepository) List(ctx context.Context) ([]User, error) {
	cursor, err := repository.collection.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{
		{Key: "created_at", Value: 1}, {Key: "_id", Value: 1},
	}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	users := make([]User, 0)
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}
