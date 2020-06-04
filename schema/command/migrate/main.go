package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/store"
)

const testingCenterFilepath = "./data/TaiwanCDCTestCenter.csv"

func init() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("autonomy")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

func main() {
	db, err := gorm.Open("postgres", viper.GetString("orm.conn"))
	if err != nil {
		panic(err)
	}

	if err := db.Exec(`CREATE SCHEMA IF NOT EXISTS autonomy`).Error; err != nil {
		panic(err)
	}

	if err := db.Exec("SET search_path TO autonomy").Error; err != nil {
		panic(err)
	}

	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(
		&schema.Account{},
		&schema.AccountProfile{},
		&schema.HelpRequest{},
	).Error; err != nil {
		panic(err)
	}

	if err := db.Model(schema.HelpRequest{}).Where(fmt.Sprintf("state = '%s'", "PENDING")).
		AddUniqueIndex("help_request_unique_if_not_done", "requester").Error; err != nil {
		panic(err)
	}

	schema.NewMongoDBIndexer(viper.GetString("mongo.conn"), viper.GetString("mongo.database")).IndexAll()

	err = migrateMongo()
	if nil != err {
		panic(err)
	}
}

func migrateMongo() error {
	ctx := context.Background()
	opts := options.Client().ApplyURI(viper.GetString("mongo.conn"))
	opts.SetMaxPoolSize(1)
	client, _ := mongo.NewClient(opts)
	_ = client.Connect(ctx)

	if err := setupCollectionBehavior(ctx, client); err != nil {
		fmt.Println("failed to set up collection `behavior`: ", err)
		return err
	}

	if err := BehaviorListNullToEmptyArray(client); err != nil {
		fmt.Println("failed to convert null to empty array in  `behavior` list : ", err)
		return err
	}

	if err := setupCollectionSymptom(ctx, client); err != nil {
		fmt.Println("failed to set up collection `symptom`: ", err)
		return err
	}

	if err := setupCDSConfirmCollection(client); err != nil {
		fmt.Println("failed to set up collection `cds confirm`: ", err)
		return err
	}
	if err := setupTestCenter(client); err != nil {
		fmt.Println("failed to set up collection `testCenter collection`: ", err)
		return err
	}

	return nil
}

func setupCollectionBehavior(ctx context.Context, client *mongo.Client) error {
	fmt.Println("initialize behavior collection")
	c := client.Database(viper.GetString("mongo.database")).Collection(schema.BehaviorCollection)

	behaviors := make([]interface{}, 0, len(schema.OfficialBehaviorMatrix))
	for _, b := range schema.OfficialBehaviors {
		behaviors = append(behaviors, b)
	}

	if _, err := c.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.M{
			"source": 1,
		},
	}); err != nil {
		return err
	}

	_, err := c.InsertMany(ctx, behaviors)
	if err != nil {
		if errs, hasErr := err.(mongo.BulkWriteException); hasErr {
			if 1 == len(errs.WriteErrors) && store.DuplicateKeyCode == errs.WriteErrors[0].Code {
				return nil
			}
		}
	}

	return err
}

func setupCollectionSymptom(ctx context.Context, client *mongo.Client) error {
	fmt.Println("initialize symptom collection")
	c := client.Database(viper.GetString("mongo.database")).Collection(schema.SymptomCollection)

	if _, err := c.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.M{
			"source": 1,
		},
	}); err != nil {
		return err
	}

	if _, err := c.DeleteMany(ctx, bson.M{"source": schema.OfficialSymptom}); err != nil {
		return err
	}

	officialSymptoms := make([]interface{}, 0, len(schema.COVID19Symptoms))
	for _, s := range schema.COVID19Symptoms {
		officialSymptoms = append(officialSymptoms, s)
	}
	if _, err := c.InsertMany(ctx, officialSymptoms); err != nil {
		return err
	}

	generalSymptoms := make([]interface{}, 0, len(schema.GeneralSymptoms))
	for _, s := range schema.GeneralSymptoms {
		generalSymptoms = append(generalSymptoms, s)
	}
	if _, err := c.InsertMany(ctx, generalSymptoms); err != nil {
		if errs, hasErr := err.(mongo.BulkWriteException); hasErr {
			if !(1 == len(errs.WriteErrors) && store.DuplicateKeyCode == errs.WriteErrors[0].Code) {
				return err
			}
		}
	}

	return nil
}

func BehaviorListNullToEmptyArray(client *mongo.Client) error {
	c := client.Database(viper.GetString("mongo.database")).Collection(schema.BehaviorReportCollection)
	filter := bson.D{{"official_behaviors", bson.M{"$type": 10}}}
	update := bson.D{{"$set", bson.D{{"official_behaviors", []schema.Behavior{}}}}}
	result, err := c.UpdateMany(context.Background(), filter, update)
	if err != nil {
		return err
	}
	fmt.Println("Migration: replace Official Behavior List with Empty Array result:", result.MatchedCount)
	filter = bson.D{{"customized_behaviors", bson.M{"$type": 10}}}
	update = bson.D{{"$set", bson.D{{"customized_behaviors", []schema.Behavior{}}}}}
	result, err = c.UpdateMany(context.Background(), filter, update)
	if err != nil {
		return err
	}
	fmt.Println("Migration: replace Customized Behavior List with Empty Array result:", result.MatchedCount)
	return nil
}

func setupCDSConfirmCollection(client *mongo.Client) error {
	db := client.Database(viper.GetString("mongo.database"))
	cdsIndex := mongo.IndexModel{
		Keys:    bson.D{{"name", 1}, {"report_ts", 1}},
		Options: options.Index().SetUnique(true),
	}
	usCol := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(schema.CdsUSA)]
	_, err := db.Collection(usCol).Indexes().CreateOne(context.Background(), cdsIndex)

	if nil != err {
		fmt.Println("collection", usCol, "mongodb create name and report_ts combined index with error: ", err)
		return err
	}
	fmt.Println("confirm collection initialized", usCol)
	twCol := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(schema.CdsTaiwan)]
	_, err = db.Collection(twCol).Indexes().CreateOne(context.Background(), cdsIndex)

	if nil != err {
		fmt.Println("collection", twCol, "mongodb create name and report_ts combined index with error: ", err)
		return err
	}
	fmt.Println("confirm collection initialized", twCol)

	icelandCol := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(schema.CdsIceland)]
	_, err = db.Collection(icelandCol).Indexes().CreateOne(context.Background(), cdsIndex)

	if nil != err {
		fmt.Println("collection", icelandCol, "mongodb create name and report_ts combined index with error: ", err)
		return err
	}
	fmt.Println("confirm collection initialized", icelandCol)

	return nil
}

func setupTestCenter(client *mongo.Client) error {
	db := client.Database(viper.GetString("mongo.database"))
	centers, err := loadTestCenter(testingCenterFilepath)
	if err != nil {
		return err
	}
	centersToInterface := make([]interface{}, 0, len(centers))
	for _, c := range centers {
		centersToInterface = append(centersToInterface, c)
	}
	db.Collection(schema.TestCenterCollection).Drop(context.Background())
	_, err = db.Collection(schema.TestCenterCollection).InsertMany(context.Background(), centersToInterface)
	if err != nil {
		return err
	}
	fmt.Println("testCenter collection initialized", schema.TestCenterCollection)
	return nil
}

func loadTestCenter(filepath string) ([]schema.TestCenter, error) {
	testingCenter, err := os.Open(filepath)
	if err != nil {
		return []schema.TestCenter{}, err
	}
	centers := []schema.TestCenter{}
	r := csv.NewReader(testingCenter)
	for {
		// Read each record from csv
		record, err := r.Read()

		if err != nil {
			if err == io.EOF {
				break
			}
			return centers, err
		}
		switch record[0] {
		case schema.CdsTaiwan:
			if len(record) < 8 {
				continue
			}
			lat, err := strconv.ParseFloat(record[4], 64)
			if err != nil {
				continue
			}
			long, err := strconv.ParseFloat(record[5], 64)
			if err != nil {
				continue
			}
			center := schema.TestCenter{
				Country:         schema.CDSCountryType(record[0]),
				County:          record[1],
				InstitutionCode: record[2],
				Location:        schema.GeoJSON{Type: "Point", Coordinates: []float64{long, lat}},
				Name:            record[3],
				Address:         record[6],
				Phone:           record[7],
			}
			centers = append(centers, center)
		}
	}
	return centers, nil
}
