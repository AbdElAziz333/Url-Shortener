package stat

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"gorm.io/gorm"
)

type Repository interface {
	GetTotalClicks(ctx context.Context, code string) ([]Dto, error)
	GetGeo(ctx context.Context, code string) ([]Dto, error)
	GetReferrers(ctx context.Context, code string) ([]Dto, error)
	SaveClickEvent(ctx context.Context, event map[string]any) error
}

type repository struct {
	db          *gorm.DB
	mongoClient *mongo.Client
}

func NewRepository(db *gorm.DB, mongoClient *mongo.Client) Repository {
	return &repository{
		db:          db,
		mongoClient: mongoClient,
	}
}

func (r *repository) GetTotalClicks(ctx context.Context, code string) ([]Dto, error) {
	logrus.WithField("code", code).Debug("Getting total clicks")
	var total int64
	err := r.db.WithContext(ctx).
		Model(&LinkStatsDaily{}).
		Select("COALESCE(SUM(click_count), 0)").
		Where("link_code = ?", code).
		Scan(&total).Error
	if err != nil {
		logrus.WithError(err).WithField("code", code).Error("Failed to get total clicks from PostgreSQL")
		return nil, err
	}

	return []Dto{{Key: "total", Count: total}}, nil
}

func (r *repository) GetGeo(ctx context.Context, code string) ([]Dto, error) {
	logrus.WithField("code", code).Debug("Getting geo stats")
	// Assumes raw click events are stored in MongoDB:
	// db: analytics, collection: click_events, fields: { code, country }
	// If your schema differs, adjust the `dbName`, `collectionName`, and field paths.
	const dbName = "analytics"
	const collectionName = "click_events"

	coll := r.mongoClient.Database(dbName).Collection(collectionName)

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{{Key: "code", Value: code}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$country", "unknown"}}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
		bson.D{{Key: "$limit", Value: 10}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "key", Value: "$_id"},
			{Key: "count", Value: 1},
		}}},
	}

	cur, err := coll.Aggregate(ctx, pipeline, options.Aggregate())
	if err != nil {
		logrus.WithError(err).WithField("code", code).Error("Failed to get geo stats from MongoDB")
		return nil, err
	}
	defer cur.Close(ctx)

	var out []Dto
	if err := cur.All(ctx, &out); err != nil {
		logrus.WithError(err).WithField("code", code).Error("Failed to decode geo stats from MongoDB")
		return nil, err
	}
	return out, nil
}

func (r *repository) GetReferrers(ctx context.Context, code string) ([]Dto, error) {
	logrus.WithField("code", code).Debug("Getting referrer stats")
	// Assumes raw click events are stored in MongoDB:
	// db: analytics, collection: click_events, fields: { code, referrer_domain | referrer }
	// If your schema differs, adjust field paths below.
	const dbName = "analytics"
	const collectionName = "click_events"

	coll := r.mongoClient.Database(dbName).Collection(collectionName)

	refKey := bson.D{{Key: "$ifNull", Value: bson.A{"$referrer_domain", "$referrer"}}}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{{Key: "code", Value: code}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$ifNull", Value: bson.A{refKey, "unknown"}}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
		bson.D{{Key: "$limit", Value: 10}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "key", Value: "$_id"},
			{Key: "count", Value: 1},
		}}},
	}

	cur, err := coll.Aggregate(ctx, pipeline, options.Aggregate())
	if err != nil {
		logrus.WithError(err).WithField("code", code).Error("Failed to get referrer stats from MongoDB")
		return nil, err
	}
	defer cur.Close(ctx)

	var out []Dto
	if err := cur.All(ctx, &out); err != nil {
		logrus.WithError(err).WithField("code", code).Error("Failed to decode referrer stats from MongoDB")
		return nil, err
	}
	return out, nil
}

func (r *repository) SaveClickEvent(ctx context.Context, event map[string]any) error {
	logrus.WithField("event", event).Debug("Saving click event")
	// 1. Insert raw event into MongoDB for Geo and Referrer stats
	const dbName = "analytics"
	const collectionName = "click_events"
	coll := r.mongoClient.Database(dbName).Collection(collectionName)
	if _, err := coll.InsertOne(ctx, event); err != nil {
		logrus.WithError(err).Warn("Failed to insert click event into MongoDB")
	}

	// 2. Extract code to update daily stats in PostgreSQL
	code, ok := event["code"].(string)
	if !ok || code == "" {
		logrus.Debug("No code in event, skipping PostgreSQL update")
		return nil // if there's no code, we can't update LinkStatsDaily
	}

	// Upsert to LinkStatsDaily (for total clicks) - assumes date is just today
	// This is a basic implementation to increment clicks
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO link_stats_daily (id, link_code, date, click_count)
		VALUES (gen_random_uuid(), ?, CURRENT_DATE, 1)
		ON CONFLICT (link_code, date) DO UPDATE 
		SET click_count = link_stats_daily.click_count + 1
	`, code).Error
	if err != nil {
		logrus.WithError(err).WithField("code", code).Error("Failed to update daily stats in PostgreSQL")
	}

	return err
}
