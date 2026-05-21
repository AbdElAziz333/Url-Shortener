package stat

import (
	"context"

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
	db            *gorm.DB
	mongoClient   *mongo.Client
}

func NewRepository(db *gorm.DB, mongoClient *mongo.Client) Repository {
	return &repository{
		db:            db,
		mongoClient:   mongoClient,
	}
}

func (r *repository) GetTotalClicks(ctx context.Context, code string) ([]Dto, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&LinkStatsDaily{}).
		Select("COALESCE(SUM(click_count), 0)").
		Where("link_code = ?", code).
		Scan(&total).Error
	if err != nil {
		return nil, err
	}

	return []Dto{{Key: "total", Count: total}}, nil
}

func (r *repository) GetGeo(ctx context.Context, code string) ([]Dto, error) {
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
		return nil, err
	}
	defer cur.Close(ctx)

	var out []Dto
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) GetReferrers(ctx context.Context, code string) ([]Dto, error) {
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
		return nil, err
	}
	defer cur.Close(ctx)

	var out []Dto
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) SaveClickEvent(ctx context.Context, event map[string]any) error {
	// 1. Insert raw event into MongoDB for Geo and Referrer stats
	const dbName = "analytics"
	const collectionName = "click_events"
	coll := r.mongoClient.Database(dbName).Collection(collectionName)
	if _, err := coll.InsertOne(ctx, event); err != nil {
		// Log but don't fail immediately, try to do postgres
		// log.Printf("warn: failed to insert into mongo: %v", err)
	}

	// 2. Extract code to update daily stats in PostgreSQL
	code, ok := event["code"].(string)
	if !ok || code == "" {
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

	return err
}
