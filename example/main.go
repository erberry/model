package main

import (
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var (
	db         *gorm.DB
	pool       *redis.Pool
	etcdClient *clientv3.Client
)

func init() {
	var err error
	db, err = gorm.Open("mysql", "user:pwd@tcp(127.0.0.1)/dbname?charset=utf8mb4&loc=Asia%2FShanghai&parseTime=true")
	if err != nil {
		panic(err)
	}

	pool = newPool("127.0.0.1:6379", "")

	etcdClient, err = clientv3.New(clientv3.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	})
	if err != nil {
		panic(err)
	}
}

func newPool(addr string, pwd string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 10 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) { return redis.Dial("tcp", addr, redis.DialPassword(pwd)) },
	}
}

func main() {
	pool.Get().Do("FLUSHDB")
	simple()
	pool.Get().Do("FLUSHDB")
	hsimple()

	list()
}
