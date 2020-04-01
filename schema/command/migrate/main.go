package main

import (
	"context"
	"fmt"
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

	db.AutoMigrate(
		&schema.Account{},
		&schema.AccountProfile{},
		&schema.HelpRequest{},
	)

	err = migrateMongo()
	if nil != err {
		panic(err)
	}
}

func migrateMongo() error {
	opts := options.Client().ApplyURI(viper.GetString("mongo.conn"))
	opts.SetMaxPoolSize(1)
	client, _ := mongo.NewClient(opts)
	_ = client.Connect(context.Background())
	c := client.Database(viper.GetString("mongo.database")).Collection(store.ProfileCollectionName)

	// here is reference from api/store/profile
	// if bson key of location is changed, here should also be changed
	idx := mongo.IndexModel{
		Keys: bson.M{
			"location": "2dsphere",
		},
		Options: nil,
	}

	_, err := c.Indexes().CreateOne(context.Background(), idx)
	if nil != err {
		fmt.Println("mongodb create index with error: ", err)
		return err
	}

	return nil
}
