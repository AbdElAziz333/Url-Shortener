package mongo

import (
	"aziz.dev/analytics/internal/config"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func NewClient(cfg *config.MongoConfig) (*mongo.Client, error) {
	logrus.Info("Connecting to MongoDB")
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.URI))
	if err != nil {
		logrus.WithError(err).Error("Failed to connect to MongoDB")
		return nil, err
	}

	return client, nil
}