package schema

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBIndexer struct {
	ctx      context.Context
	dbName   string
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDBIndexer(connectionString, dbName string) *MongoDBIndexer {
	ctx := context.Background()
	opts := options.Client().ApplyURI(connectionString)
	client, err := mongo.NewClient(opts)
	if err != nil {
		panic(err)
	}
	if err := client.Connect(ctx); err != nil {
		panic(err)
	}

	return &MongoDBIndexer{
		ctx:      ctx,
		dbName:   dbName,
		Client:   client,
		Database: client.Database(dbName),
	}
}

func (m *MongoDBIndexer) createIndex(collection string, index mongo.IndexModel) error {
	c := m.Database.Collection(collection)
	_, err := c.Indexes().CreateOne(m.ctx, index)
	return err
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func (m *MongoDBIndexer) IndexAll() {
	panicIfError(m.IndexProfileCollection())
	panicIfError(m.IndexBoundryCollection())
	panicIfError(m.IndexPOICollection())
	panicIfError(m.IndexBehaviorCollection())
	panicIfError(m.IndexBehaviorReportCollection())
	panicIfError(m.IndexSymptomCollection())
	panicIfError(m.IndexSymptomReportCollection())
	panicIfError(m.IndexCDSConfirmCollection())
	panicIfError(m.IndexGuideCollection())
}

func (m *MongoDBIndexer) IndexProfileCollection() error {
	if err := m.createIndex(ProfileCollection, mongo.IndexModel{
		Keys: bson.M{
			"id": 1,
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	m.createIndex(ProfileCollection, mongo.IndexModel{
		Keys: bson.M{
			"account_number": 1,
		},
		Options: options.Index().SetUnique(true),
	})

	return m.createIndex(ProfileCollection, mongo.IndexModel{
		Keys: bson.M{
			"location": "2dsphere",
		},
	})
}

func (m *MongoDBIndexer) IndexPOICollection() error {
	return m.createIndex(POICollection, mongo.IndexModel{
		Keys: bson.M{
			"location": "2dsphere",
		},
	})
}

func (m *MongoDBIndexer) IndexBehaviorCollection() error {
	return m.createIndex(BehaviorCollection, mongo.IndexModel{
		Keys: bson.M{
			"source": 1,
		},
	})
}

func (m *MongoDBIndexer) IndexBehaviorReportCollection() error {
	if err := m.createIndex(BehaviorReportCollection, mongo.IndexModel{
		Keys: bson.D{
			{"profile_id", 1},
			{"ts", 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	return m.createIndex(BehaviorReportCollection, mongo.IndexModel{
		Keys: bson.M{
			"location": "2dsphere",
		},
	})
}

func (m *MongoDBIndexer) IndexSymptomCollection() error {
	return m.createIndex(SymptomCollection, mongo.IndexModel{
		Keys: bson.M{
			"source": 1,
		},
	})
}

func (m *MongoDBIndexer) IndexSymptomReportCollection() error {
	if err := m.createIndex(SymptomReportCollection, mongo.IndexModel{
		Keys: bson.D{
			{"profile_id", 1},
			{"ts", 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	return m.createIndex(SymptomReportCollection, mongo.IndexModel{
		Keys: bson.M{
			"location": "2dsphere",
		},
	})
}

func (m *MongoDBIndexer) IndexBoundryCollection() error {
	if err := m.createIndex(BoundaryCollection, mongo.IndexModel{
		Keys: bson.M{
			"county":  1,
			"island":  1,
			"country": 1,
			"state":   1,
		},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	if err := m.createIndex(BoundaryCollection, mongo.IndexModel{
		Keys: bson.M{
			"country": 1,
		},
	}); err != nil {
		return err
	}

	return m.createIndex(BoundaryCollection, mongo.IndexModel{
		Keys: bson.M{
			"geometry": "2dsphere",
		},
	})
}

func (m *MongoDBIndexer) IndexCDSConfirmCollection() error {
	cdsIndex := mongo.IndexModel{
		Keys:    bson.D{{"name", 1}, {"report_ts", 1}},
		Options: options.Index().SetUnique(true),
	}
	usCollection := CDSCountyCollectionMatrix[CDSCountryType(CdsUSA)]
	if err := m.createIndex(usCollection, cdsIndex); err != nil {
		return err
	}
	twCollection := CDSCountyCollectionMatrix[CDSCountryType(CdsTaiwan)]
	if err := m.createIndex(twCollection, cdsIndex); err != nil {
		return err
	}
	icelandCollection := CDSCountyCollectionMatrix[CDSCountryType(CdsIceland)]
	if err := m.createIndex(icelandCollection, cdsIndex); err != nil {
		return err
	}
	return nil
}
func (m *MongoDBIndexer) IndexGuideCollection() error {
	return m.createIndex(TestCenterCollection, mongo.IndexModel{
		Keys: bson.M{
			"location": "2dsphere",
		},
	})
}
