package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/go-redis/redis/v8"
	"github.com/influxdata/influxdb-client-go/v2"
	"gorm.io/gorm"
	"github.com/weeon/contract"
	"github.com/weeon/log"
	"github.com/weeon/mod"
	"go.mongodb.org/mongo-driver/mongo"
	mgopt "go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
)

type ServiceConfig struct {
	Namespace       string
	Service         string
	ServiceKey      string
	HttpPort        int
	IP              string
	HealthCheckPath string
}

type ServiceOpt struct {

}

type App struct {
	serviceConfig ServiceConfig
	config        contract.Config
	registerFn    RegisterFn
	conf          *Config

	teardownFns []func()

	confKeys []string

	//
	dbKeys       []string
	redisKeys    []string
	mongoKeys    []string
	influxdbKeys []string

	db       map[string]*gorm.DB
	redis    map[string]*redis.Client
	mongo    map[string]*mongo.Client
	influxDB map[string]influxdb2.Client

	grpcConn map[string]string
}

const (
	Comm = "comm"
	Srv  = "srv"
)

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
	GrpcConn map[string]string
}

func NewConfig() *Config {
	return &Config{
		Database: make(map[string]mod.Database),
		Redis:    make(map[string]mod.Redis),
		Mongo:    make(map[string]mod.Mongo),
		InfluxDB: make(map[string]mod.InfluxDB),
	}
}

func NewApp(c contract.Config, registerFn RegisterFn, cfg ServiceConfig) (*App, error) {
	app := App{
		serviceConfig: ServiceConfig{
			Namespace:       cfg.Namespace,
			Service:         cfg.Service,
			ServiceKey:      fmt.Sprintf("%s/%s", cfg.Namespace, cfg.Service),
			HttpPort:        cfg.HttpPort,
			IP:              cfg.IP,
			HealthCheckPath: cfg.HealthCheckPath,
		},
		config:      c,
		conf:        NewConfig(),
		registerFn:  registerFn,
		teardownFns: make([]func(), 0),

		confKeys: make([]string, 0),

		dbKeys:       make([]string, 0),
		redisKeys:    make([]string, 0),
		mongoKeys:    make([]string, 0),
		influxdbKeys: make([]string, 0),

		db:       map[string]*gorm.DB{},
		redis:    map[string]*redis.Client{},
		mongo:    map[string]*mongo.Client{},
		influxDB: map[string]influxdb2.Client{},

		grpcConn: make(map[string]string),
	}
	return &app, nil
}

func (a *App) Register() error {
	fn, err := a.registerFn(a.serviceConfig)
	if err != nil {
		return err
	}
	a.teardownFns = append(a.teardownFns, fn)
	return nil
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

func (a *App) AddInfluxDBKey(ks ...string) {
	a.influxdbKeys = append(a.influxdbKeys, ks...)
}

func (a *App) genConfKey(dir, k string) string {
	return fmt.Sprintf("%s/config/%s/%s", a.serviceConfig.Namespace, dir, k)
}

func (a *App) InitConf() error {
	var configKeys = map[string]interface{}{
		Database: &a.conf.Database,
		Redis:    &a.conf.Redis,
		Mongo:    &a.conf.Mongo,
		InfluxDB: &a.conf.InfluxDB,
		GrpcConn: &a.conf.GrpcConn,
	}

	for _, v := range a.confKeys {
		if vv, ok := configKeys[v]; ok {
			if err := a.CommConfDecode(v, vv); err != nil {
				return err
			}
			continue
		}
		return errors.New(fmt.Sprintf("key %s not found", v))
	}
	return nil
}

func (a *App) SrvConfDecode(name string, conf interface{}) error {
	return a.ConfDecode(Srv, name, conf)
}

func (a *App) CommConfDecode(name string, conf interface{}) error {
	return a.ConfDecode(Comm, name, conf)
}

func (a *App) ConfDecode(dir, name string, conf interface{}) error {
	b, err := a.config.Get(a.genConfKey(dir, name))
	if err != nil {
		return err
	}
	err = toml.Unmarshal(b, conf)
	if err != nil {
		return err
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
		cli := influxdb2.NewClient(vv.Addr, vv.Token)
		a.influxDB[v] = cli
	}
	return nil
}

func (a *App) GetInfluxDB(k string) influxdb2.Client {
	return a.influxDB[k]
}

func (a *App) GetRedis(k string) *redis.Client {
	return a.redis[k]
}

func (a *App) InitMongo() error {
	for _, v := range a.mongoKeys {
		vv, ok := a.conf.Mongo[v]
		if !ok {
			return errors.New(fmt.Sprintf("mongo key %s not found", v))
		}
		cli, err := mongo.NewClient(mgopt.Client().ApplyURI(vv.Uri))
		if err != nil {
			return err
		}
		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
		err = cli.Connect(ctx)
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

	engine, err := gorm.Open(mysql.Open(newConnStr(m)), &gorm.Config{} )
	return engine, err
}

func (a *App) dialGrpcConn(addr string) *grpc.ClientConn {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Errorf("get grpc conn error addr %s error: %v", addr, err)
	}
	return conn
}

func (a *App) GetGrpcConn(name string) *grpc.ClientConn {
	addr := a.conf.GrpcConn[name]
	return a.dialGrpcConn(addr)
}
