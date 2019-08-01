package app

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/BurntSushi/toml"
	"github.com/go-redis/redis"
	"github.com/influxdata/influxdb-client-go"
	"github.com/jinzhu/gorm"
	"github.com/weeon/contract"
	"github.com/weeon/mod"
	"go.mongodb.org/mongo-driver/mongo"
	mgopt "go.mongodb.org/mongo-driver/mongo/options"
)

type App struct {
	namespace string
	config    contract.Config
	conf      *Config

	confKeys []string

	//
	dbKeys       []string
	redisKeys    []string
	mongoKeys    []string
	influxdbKeys []string

	db       map[string]*gorm.DB
	redis    map[string]*redis.Client
	mongo    map[string]*mongo.Client
	influxDB map[string]*influxdb.Client

	grpcConn map[string]string
}

const (
	Database = "database"
	Redis    = "redis"
	Mongo    = "mongo"
	InfluxDB = "influxdb"
	GrpcConn = "grpc_conn"
)

type Config struct {
	Database map[string]mod.Database
	Redis    map[string]mod.Redis
	Mongo    map[string]mod.Mongo
	InfluxDB map[string]mod.InfluxDB
}

func NewConfig() *Config {
	return &Config{
		Database: make(map[string]mod.Database),
		Redis:    make(map[string]mod.Redis),
		Mongo:    make(map[string]mod.Mongo),
		InfluxDB: make(map[string]mod.InfluxDB),
	}
}

func NewApp(namespace string, c contract.Config) (*App, error) {
	app := App{
		namespace: namespace,
		config:    c,
		conf:      NewConfig(),

		confKeys: make([]string, 0),

		dbKeys:       make([]string, 0),
		redisKeys:    make([]string, 0),
		mongoKeys:    make([]string, 0),
		influxdbKeys: make([]string, 0),

		db:       map[string]*gorm.DB{},
		redis:    map[string]*redis.Client{},
		mongo:    map[string]*mongo.Client{},
		influxDB: map[string]*influxdb.Client{},

		grpcConn: make(map[string]string),
	}
	return &app, nil
}

func (a *App) AddConfKey(ks ...string) {
	a.confKeys = append(a.confKeys, ks...)
}

func (a *App) AddDatabaseKey(ks ...string) {
	a.dbKeys = append(a.dbKeys, ks...)
}

func (a *App) AddRedisKey(ks ...string) {
	a.redisKeys = append(a.redisKeys, ks...)
}

func (a *App) AddMongoKey(ks ...string) {
	a.mongoKeys = append(a.mongoKeys, ks...)
}

func (a *App) AddInfluxdbKey(ks ...string) {
	a.influxdbKeys = append(a.influxdbKeys, ks...)
}

func (a *App) genConfKey(k string) string {
	return fmt.Sprintf("%s/config/%s", a.namespace, k)
}

func (a *App) InitConf() error {
	var configKeys = map[string]interface{}{
		Database: &a.conf.Database,
		Redis:    &a.conf.Redis,
		Mongo:    &a.conf.Mongo,
		InfluxDB: &a.conf.InfluxDB,
	}

	for _, v := range a.confKeys {
		if vv, ok := configKeys[v]; ok {
			b, err := a.config.Get(a.genConfKey(v))
			if err != nil {
				return err
			}
			err = toml.Unmarshal(b, vv)
			if err != nil {
				return err
			}
			continue
		}
		return errors.New(fmt.Sprintf("key %s not found", v))
	}
	return nil
}

func (a *App) InitDatabase() error {
	for _, v := range a.dbKeys {
		if vv, ok := a.conf.Database[v]; ok {
			db, err := NewDatabase(vv)
			if err != nil {
				return err
			}
			a.db[v] = db
			continue
		}
		return errors.New(fmt.Sprintf("db config %s not found", v))
	}
	return nil
}

func (a *App) GetDB(k string) *gorm.DB {
	return a.db[k]
}

func (a *App) InitRedis() error {
	for _, v := range a.redisKeys {
		vv, ok := a.conf.Redis[v]
		if !ok {
			return errors.New(fmt.Sprintf("redis key %s not found", v))
		}
		cli := redis.NewClient(&redis.Options{
			Addr:     vv.Host,
			DB:       vv.DB,
			Password: vv.Password,
		})
		a.redis[v] = cli
	}
	return nil
}

func (a *App) InitInfluxDB() error {
	for _, v := range a.influxdbKeys {
		vv, ok := a.conf.InfluxDB[v]
		if !ok {
			return errors.New(fmt.Sprintf("influxdb key %s not found", v))
		}
		cli, err := influxdb.New(http.DefaultClient, influxdb.WithAddress(vv.Addr),
			influxdb.WithUserAndPass(vv.Username, vv.Password))
		if err != nil {
			return err
		}
		a.influxDB[v] = cli
	}
	return nil
}

func (a *App) GetRedis(k string) *redis.Client {
	return a.redis[k]
}

func (a *App) InitMongo(k string) error {
	for _, v := range a.mongoKeys {
		vv, ok := a.conf.Mongo[v]
		if !ok {
			return errors.New(fmt.Sprintf("mongo key %s not found", v))
		}
		cli, err := mongo.NewClient(mgopt.Client().ApplyURI(vv.Uri))
		if err != nil {
			return err
		}
		a.mongo[v] = cli
	}
	return nil
}

func (a *App) GetMongo(k string) *mongo.Client {
	return a.mongo[k]
}

func newConnStr(m mod.Database) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		m.User, m.Password, m.Host, m.Port, m.Database)
}

func NewDatabase(m mod.Database) (*gorm.DB, error) {
	var err error
	engine, err := gorm.Open(m.Driver, newConnStr(m))
	return engine, err
}
