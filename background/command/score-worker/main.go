package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	scoreWorker "github.com/bitmark-inc/autonomy-api/background/score"
	cadence "github.com/bitmark-inc/autonomy-api/external/cadence"
	"github.com/bitmark-inc/autonomy-api/geo"
	"github.com/bitmark-inc/autonomy-api/store"
)

var logger *zap.Logger

func init() {
	logger = buildLogger()
}

func buildLogger() *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.Level.SetLevel(zapcore.InfoLevel)

	var err error
	logger, err := config.Build()
	if err != nil {
		panic("Failed to setup logger")
	}

	return logger
}

func initSentry() {
	// Sentry
	logger.Info("Initializing sentry")
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              viper.GetString("sentry.dsn"),
		AttachStacktrace: true,
		Environment:      viper.GetString("sentry.environment"),
		Dist:             viper.GetString("sentry.dist"),
	}); err != nil {
		logger.Panic("fail to initialize sentry", zap.Error(err))
	}
}

func loadConfig(file string) {
	// Config from file
	viper.SetConfigType("yaml")
	if file != "" {
		viper.SetConfigFile(file)
	}

	viper.AddConfigPath("/.config/")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("No config file. Read config from env.")
		viper.AllowEmptyEnv(false)
	}

	// Config from env if possible
	viper.AutomaticEnv()
	viper.SetEnvPrefix("autonomy")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "./config.yaml", "[optional] path of configuration file")
	flag.Parse()

	loadConfig(configFile)
	initSentry()

	opts := options.Client().ApplyURI(viper.GetString("mongo.conn"))
	opts.SetMaxPoolSize(viper.GetUint64("mongo.pool"))
	mongoClient, err := mongo.NewClient(opts)
	if nil != err {
		logger.Panic("create mongo client with error", zap.Error(err))
	}

	err = mongoClient.Connect(context.Background())
	if nil != err {
		logger.Panic("connect mongo database with error", zap.Error(err))
	}

	geo.SetLocationResolver(geo.NewMongodbLocationResolver(mongoClient, viper.GetString("mongo.database")))

	mongoStore := store.NewMongoStore(
		mongoClient,
		viper.GetString("mongo.database"),
	)

	worker := scoreWorker.NewScoreUpdateWorker(viper.GetString("cadence.domain"), mongoStore)
	worker.Register()
	worker.Start(cadence.BuildCadenceServiceClient(viper.GetString("cadence.conn")), logger)
}
