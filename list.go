package model

import (
	"errors"
	"math/rand"
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
	Load(offset, limit int) ([]Z, error)

	//RedisListKey
	RedisListKey() string
	//RedisStub
	RedisStub() RedisStub

	//GetLocker
	GetLocker() Locker
}

//GetByPage GetByPage
func GetByPage(l List, offset, limit int, reverse bool) ([]Z, int, error) {
	key := l.RedisListKey()
	stub := l.RedisStub()

	exist, err := redis.Int(stub.Do("EXISTS", key))
	if err != nil {
		return []Z{}, 0, err
	}
	if exist != 1 {
		//load
		err := loadList(l, true)
		if err != nil {
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
func UpdateList(l List, z Z) error {
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
	return err
}

//ReloadList ReloadList
func ReloadList(l List) error {
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

	return loadList(l, false)
}

//loadList loadList
func loadList(l List, needLock bool) error {
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

	offset := 0
	limit := 1000
	for {
		zs, err := l.Load(offset, limit)
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

	total, _ := redis.Int(stub.Do("ZCARD", key))
	expire := determinExpire(int(total))
	stub.Do("EXPIRE", key, int(expire.Seconds()))

	return nil
}

func determinExpire(pCount int) time.Duration {
	if pCount < 100 {
		return 1 * time.Hour
	} else if pCount < 1000 {
		return 24 * time.Hour
	} else if pCount < 10000 {
		return time.Duration(7*24+rand.Intn(7*12)) * time.Hour
	} else if pCount < 50000 {
		return time.Duration(14*24+rand.Intn(7*12)) * time.Hour
	} else {
		return time.Duration(30*24+rand.Intn(7*12)) * time.Hour
	}
}
