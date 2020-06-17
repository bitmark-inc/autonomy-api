package main

import (
	"context"
	"strings"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/share/geojson"
)

func init() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("autonomy")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

func main() {
	ctx := context.Background()
	opts := options.Client().ApplyURI(viper.GetString("mongo.conn"))
	client, err := mongo.NewClient(opts)
	if err != nil {
		panic(err)
	}
	if err := client.Connect(ctx); err != nil {
		panic(err)
	}

	dbName := viper.GetString("mongo.database")

	if err := geojson.ImportTaiwanBoundary(client, dbName, "tw-boundary.json"); err != nil {
		panic(err)
	}

	if err := geojson.ImportWorldCountryBoundary(client, dbName, "world-boundary.geojson"); err != nil {
		panic(err)
	}

	if err := geojson.ImportUSBoundary(client, dbName, "us-boundary.geojson"); err != nil {
		panic(err)
	}
}
