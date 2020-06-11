package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"googlemaps.github.io/maps"

	"github.com/dgrijalva/jwt-go"
	"github.com/getsentry/sentry-go"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/bitmark-inc/autonomy-api/api"
	"github.com/bitmark-inc/autonomy-api/external/aqi"
	"github.com/bitmark-inc/autonomy-api/geo"
	"github.com/bitmark-inc/autonomy-api/utils"

	bitmarksdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/bitmark-sdk-go/account"
)

var (
	server *api.Server
	ormDB  *gorm.DB
)

func initLog() {
	logLevel, err := log.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(logLevel)
	}

	log.SetOutput(os.Stdout)

	log.SetFormatter(&prefixed.TextFormatter{
		ForceFormatting: true,
		FullTimestamp:   true,
	})
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

	initialCtx, cancelInitialization := context.WithCancel(context.Background())

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info("Server is preparing to shutdown")

		if initialCtx != nil && cancelInitialization != nil {
			log.Info("Cancelling initialization")
			cancelInitialization()
			<-initialCtx.Done()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if server != nil {
			log.Info("Shutdown mobile api server")
			if err := server.Shutdown(ctx); err != nil {
				log.Error("Server Shutdown:", err)
			}
		}

		if ormDB != nil {
			log.Info("Shutting down db store")
			if err := ormDB.Close(); err != nil {
				log.Error(err)
			}
		}

		os.Exit(1)
	}()

	flag.StringVar(&configFile, "c", "./config.yaml", "[optional] path of configuration file")
	flag.Parse()

	loadConfig(configFile)

	initLog()

	utils.InitI18NBundle()

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Sentry
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              viper.GetString("sentry.dsn"),
		AttachStacktrace: true,
		Environment:      viper.GetString("sentry.environment"),
		Dist:             viper.GetString("sentry.dist"),
	}); err != nil {
		log.Error(err)
	}
	log.WithField("prefix", "init").Info("Initialized sentry")

	// Init Bitmark SDK
	bitmarksdk.Init(&bitmarksdk.Config{
		Network:    bitmarksdk.Network(viper.GetString("bitmarksdk.network")),
		APIToken:   viper.GetString("bitmarksdk.token"),
		HTTPClient: httpClient,
	})
	log.WithField("prefix", "init").Info("Initialized bitmark sdk")

	// Load global bitmark account
	a, err := account.FromSeed(viper.GetString("account.seed"))
	if err != nil {
		log.Panic(err)
	}
	globalAccount := a.(*account.AccountV2)
	log.WithField("prefix", "init").Info("Global account: ", globalAccount.AccountNumber())
	log.WithField("prefix", "init").Info("Global enc pub key: ", hex.EncodeToString(globalAccount.EncrKey.PublicKeyBytes()))

	// Load JWT private key
	jwtSecretByte, err := ioutil.ReadFile(viper.GetString("jwt.keyfile"))
	if err != nil {
		log.Panic(err)
	}
	jwtPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEMWithPassword(jwtSecretByte, viper.GetString("jwt.password"))
	if err != nil {
		log.Panic(err)
	}
	log.WithField("prefix", "init").Info("Loaded global jwt key")

	ormDB, err = gorm.Open("postgres", viper.GetString("orm.conn"))
	if err != nil {
		log.Panic(err)
	}

	// initialise mongodb connections
	opts := options.Client().ApplyURI(viper.GetString("mongo.conn"))
	opts.SetMaxPoolSize(viper.GetUint64("mongo.pool"))
	mongoClient, err := mongo.NewClient(opts)
	if nil != err {
		log.Panicf("create mongo client with error: %s", err)
	}

	err = mongoClient.Connect(context.Background())
	if nil != err {
		log.Panicf("connect mongo database with error: %s", err)
	}

	mapClient, err := maps.NewClient(maps.WithAPIKey(viper.GetString("map.key")))
	if err != nil {
		log.Panicf("init goolge map client with error: %s", err.Error())
	}

	geo.SetLocationResolver(
		geo.NewMultipleLocationResolver(
			geo.NewMongodbLocationResolver(mongoClient, viper.GetString("mongo.database")),
			geo.NewGeocodingLocationResolver(mapClient),
		),
	)

	geo.SetLocationSearcher(geo.NewNominatimSearcher(viper.GetString("nominatim.endpoint")))

	aqiClient := aqi.New(viper.GetString("aqi.key"), "")

	// Init http server
	server = api.NewServer(
		ormDB,
		mongoClient,
		jwtPrivateKey,
		globalAccount,
		aqiClient)
	log.WithField("prefix", "init").Info("Initialized http server")

	// Remove initial context
	initialCtx = nil
	cancelInitialization = nil

	log.Fatal(server.Run(":" + viper.GetString("server.port")))
}
