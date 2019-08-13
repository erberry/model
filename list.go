package model

import (
	"context"
	"errors"
	"time"

	"github.com/gomodule/redigo/redis"
)

//Z Z
type Z struct {
	Field string
	Score string
}

//List List
type List interface {
	//分页加载方法
	Load(ctx context.Context, offset, limit int) ([]Z, error)

	//RedisListKey
	RedisListKey() string
	//RedisExpire
	RedisExpire() time.Duration
	//RedisStub
	RedisStub() RedisStub
	//ListLen
	ListLen() int

	//GetLocker
	GetLocker() Locker
}

var (
	ErrOutOfRange = errors.New("out of range")
)

//GetByPage GetByPage
func GetByPage(ctx context.Context, l List, offset, limit int, reverse bool) ([]Z, int, error) {
	if l.ListLen() > 0 && offset > l.ListLen() {
		return []Z{}, 0, ErrOutOfRange
	}

	key := l.RedisListKey()
	stub := l.RedisStub()

	exist, err := redis.Int(stub.Do("EXISTS", key))
	if err != nil {
		return []Z{}, 0, err
	}
	if exist != 1 {
		//load
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := loadList(ctx, l, true)
		if err != nil {
			log.Error("load list of " + key + " failed")
			return []Z{}, 0, err
		}
	}

	start := offset
	stop := start + limit - 1
	cmd := "ZRANGE"
	if reverse {
		cmd = "ZREVRANGE"
	}

	reply, err := redis.Values(stub.Do(cmd, key, start, stop, "WITHSCORES"))
	if err != nil {
		return []Z{}, 0, err
	}
	if reply == nil || len(reply)%2 != 0 {
		return []Z{}, 0, errors.New("reply == nil || len(reply)%2 != 0")
	}

	zs := make([]Z, 0, len(reply)/2)
	for len(reply) > 0 {
		z := Z{}
		reply, err = redis.Scan(reply, &z.Field, &z.Score)
		if err != nil {
			return []Z{}, 0, err
		}
		zs = append(zs, z)
	}

	total, err := redis.Int(stub.Do("ZCARD", key))
	if err != nil {
		return []Z{}, 0, err
	}

	return zs, total, nil
}

//UpdateList UpdateList
func UpdateList(ctx context.Context, l List, z Z) error {
	key := l.RedisListKey()
	stub := l.RedisStub()
	exist, err := redis.Int(stub.Do("EXISTS", key))
	if err != nil {
		return err
	}
	if exist != 1 {
		return nil
	}

	_, err = stub.Do("ZADD", key, z.Score, z.Field)
	if l.ListLen() > 0 {
		total, err := redis.Int(stub.Do("ZCARD", key))
		if err == nil && total > l.ListLen() {
			stub.Do("ZREMRANGEBYRANK", key, 0, total-l.ListLen()-1)
		}
	}
	return err
}

//Rem Rem
func Rem(ctx context.Context, l List, field string) error {
	key := l.RedisListKey()
	stub := l.RedisStub()
	exist, err := redis.Int(stub.Do("EXISTS", key))
	if err != nil {
		return err
	}
	if exist != 1 {
		return nil
	}

	_, err = stub.Do("ZREM", key, field)
	return err
}

//ReloadList ReloadList
func ReloadList(ctx context.Context, l List) error {
	key := l.RedisListKey()
	stub := l.RedisStub()

	//lock
	locker := l.GetLocker()
	err := locker.Lock()
	if err != nil {
		return err
	}
	defer locker.Unlock()

	_, err = stub.Do("DEL", key)
	if err != nil {
		return err
	}

	err = loadList(ctx, l, false)
	if err != nil {
		log.Error("load list of " + key + " failed")
	}
	return err
}

//loadList loadList
func loadList(ctx context.Context, l List, needLock bool) error {
	key := l.RedisListKey()
	stub := l.RedisStub()

	if needLock {
		locker := l.GetLocker()
		err := locker.Lock()
		if err != nil {
			return err
		}
		defer locker.Unlock()
	}

	exist, err := redis.Int(stub.Do("EXISTS", key))
	if err != nil {
		return err
	}
	if exist == 1 {
		return nil
	}

	log.Info("start load list of " + key)

	offset := 0
	limit := 300
	for {
		if l.ListLen() > 0 && offset >= l.ListLen() {
			break
		}
		zs, err := l.Load(ctx, offset, limit)
		if err != nil {
			return err
		}

		if len(zs) == 0 {
			break
		}
		offset += limit

		args := make([]interface{}, 0, len(zs)*2+1)
		args = append(args, key)
		for _, z := range zs {
			args = append(args, z.Score, z.Field)
		}

		_, err = stub.Do("ZADD", args...)
		if err != nil {
			return err
		}
	}

	expire := l.RedisExpire()
	if expire > 0 {
		stub.Do("EXPIRE", key, int(expire.Seconds()))
	}

	log.Info("end load list of " + key)

	return nil
}
