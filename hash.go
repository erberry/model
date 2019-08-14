package model

import (
	"context"
	"fmt"
	"reflect"

	goRedis "github.com/go-redis/redis"
	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
)

//HMGet Where接口返回的条件查询 fields为空执行HGETALL
func HMGet(ctx context.Context, m Model, fields []string, fromCache bool) (notFound bool, err error) {
	if !fromCache {
		return load(m, fields)
	}

	stub := m.RedisStub()
	key := m.RedisKey(nil)

	var (
		values []interface{}
	)

	if fields != nil && len(fields) > 0 {
		args := make([]interface{}, 0, len(fields)+1)
		args = append(args, key)
		args = append(args, emptyHashField)
		args = append(args, mustExistField)
		for _, f := range fields {
			args = append(args, f)
		}

		values, err = redis.Values(stub.Do("HMGET", args...))
		if err == redis.ErrNil || err == goRedis.Nil {
			return flushHCache(ctx, m)
		}
		if s, ok := values[0].(string); ok && s == emptyHashValue {
			return true, nil
		}
		if mustExist, ok := values[1].(string); !ok || mustExist != mustExistValue {
			return flushHCache(ctx, m)
		}
		values = values[2:]
	} else {
		values, err = redis.Values(stub.Do("HGETALL", key))
		if err == redis.ErrNil || err == goRedis.Nil ||
			len(values) == 0 {
			return flushHCache(ctx, m)
		}

		if values != nil && len(values) >= 2 {
			if s, ok := values[0].(string); ok && s == emptyHashField {
				return true, nil
			}
		}
	}

	if err != nil {
		return false, err
	}

	if fields != nil && len(fields) > 0 {
		err = ScanStruct(values, m, fields)
	} else {
		err = ScanStruct(values, m, nil)
	}

	if err != nil {
		return false, err
	}

	return false, nil
}

//BatchHMGet Where接口返回的条件查询 fields为空执行HGETALL
func BatchHMGet(ctx context.Context, m Model, ids interface{}, fields []string, fromCache bool) (interface{}, error) {
	if !fromCache {
		return batchLoad(m, ids)
	}

	stub := m.RedisStub()
	v := reflect.ValueOf(ids)
	getAll := fields == nil || len(fields) == 0

	for i := 0; i < v.Len(); i++ {
		key := m.RedisKey(v.Index(i).Interface())

		if !getAll {
			args := make([]interface{}, 0, len(fields)+1)
			args = append(args, key)
			args = append(args, emptyHashField)
			args = append(args, mustExistField)
			for _, f := range fields {
				args = append(args, f)
			}

			err := stub.Send("HMGET", args...)
			if err != nil {
				return nil, err
			}
		} else {
			err := stub.Send("HGETALL", key)
			if err != nil {
				return nil, err
			}
		}
	}

	err := stub.Flush()
	if err != nil {
		return nil, err
	}

	noCached := reflect.MakeSlice(reflect.TypeOf(ids), 0, 0)
	elemType := reflect.ValueOf(m).Type()
	cachedResults := reflect.MakeSlice(reflect.SliceOf(elemType), 0, 0)

	for i := 0; i < v.Len(); i++ {
		values, err := redis.Values(stub.Receive())
		if err != nil {
			return nil, err
		}
		if !getAll {
			if err == redis.ErrNil || err == goRedis.Nil {
				noCached = reflect.Append(noCached, v.Index(i))
				continue
			}
			if s, ok := values[0].(string); ok && s == emptyHashValue {
				continue
			}
			if mustExist, ok := values[1].(string); !ok || mustExist != mustExistValue {
				noCached = reflect.Append(noCached, v.Index(i))
				continue
			}
			values = values[2:]
		} else {
			if err == redis.ErrNil || err == goRedis.Nil ||
				len(values) == 0 {
				noCached = reflect.Append(noCached, v.Index(i))
				continue
			}

			if values != nil && len(values) >= 2 {
				if s, ok := values[0].(string); ok && s == emptyHashField {
					continue
				}
			}
		}

		elem := reflect.New(elemType)
		if !getAll {
			err = ScanStruct(values, elem.Interface(), fields)
		} else {
			err = ScanStruct(values, elem.Interface(), nil)
		}

		if err != nil {
			return nil, err
		}
		cachedResults = reflect.Append(cachedResults, elem.Elem())
	}

	if noCached.Len() > 0 {
		noCachedResults, err := multiFlushHCache(ctx, m, noCached.Interface())
		if err != nil {
			return cachedResults.Interface(), err
		}

		return reflect.AppendSlice(cachedResults, reflect.ValueOf(noCachedResults)).Interface(), nil
	}

	return cachedResults.Interface(), nil
}

//HUpdate Where接口返回的条件Update。如果fields为空，更新所有字段
func HUpdate(ctx context.Context, m Model, fields map[string]interface{}) error {
	err := update(m, fields)
	stub := m.RedisStub()
	if stub != nil && err == nil {
		e, err := redis.Int(stub.Do("EXISTS", m.RedisKey(nil)))
		if err != nil || e != 1 {
			return err
		}

		if fields != nil && len(fields) > 0 {
			_, err = stub.Do("HMSET", redis.Args{}.Add(m.RedisKey(nil)).AddFlat(fields)...)
		} else {
			_, err = stub.Do("HMSET", redis.Args{}.Add(m.RedisKey(nil)).AddFlat(m)...)
		}
	}
	return err
}

//HIncr HIncr
func HIncr(ctx context.Context, m Model, fields map[string]interface{}) error {
	err := incr(m, fields)
	stub := m.RedisStub()
	if stub != nil && err == nil {
		e, err := redis.Int(stub.Do("EXISTS", m.RedisKey(nil)))
		if err != nil || e != 1 {
			return err
		}

		for field, delta := range fields {
			_, err = stub.Do("HINCRBY", m.RedisKey(nil), field, delta)
		}
	}

	return err
}

func flushHCache(ctx context.Context, m Model) (bool, error) {
	noRecord, err := load(m, nil)
	if err != nil {
		return false, err
	}

	stub := m.RedisStub()
	key := m.RedisKey(nil)

	if noRecord {
		_, err = stub.Do("HSET", key, emptyHashField, emptyHashValue)
	} else {
		_, err = stub.Do("HMSET", redis.Args{}.Add(key).AddFlat(m).
			AddFlat(map[string]interface{}{mustExistField: mustExistValue})...)
	}

	if err != nil {
		return false, err
	}

	if noRecord {
		stub.Do("EXPIRE", key, nilCacheExpire)
	} else {
		expire := int(m.RedisExpire().Seconds())
		if expire <= 0 {
			expire = defaultCacheExpire
		}
		stub.Do("EXPIRE", key, expire)
	}
	return noRecord, nil
}

func incr(m Model, fields map[string]interface{}) error {
	db := m.Where(nil)

	cols := make(map[string]interface{}, len(fields))
	for field, delta := range fields {
		cols[field] = gorm.Expr(fmt.Sprintf("%s + ?", field), delta)
	}

	err := db.Model(m).UpdateColumns(cols).Error
	return err
}

func multiFlushHCache(ctx context.Context, m Model, ids interface{}) (interface{}, error) {
	loaded, err := batchLoad(m, ids)
	if err != nil {
		return nil, err
	}

	stub := m.RedisStub()
	expire := int(m.RedisExpire().Seconds())
	if expire <= 0 {
		expire = defaultCacheExpire
	}

	v := reflect.ValueOf(loaded)
	if v.Len() == 0 {
		return loaded, nil
	}

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface().(Model)
		key := item.RedisKey(nil)
		err = stub.Send("HMSET", redis.Args{}.Add(key).AddFlat(item).
			AddFlat(map[string]interface{}{mustExistField: mustExistValue})...)
		if err != nil {
			return nil, err
		}
	}

	err = stub.Flush()
	if err != nil {
		return nil, err
	}

	for i := 0; i < v.Len(); i++ {
		_, err = stub.Receive()
		if err != nil {
			return nil, err
		}
	}

	v = reflect.ValueOf(ids)
	for i := 0; i < v.Len(); i++ {
		key := m.RedisKey(v.Index(i).Interface())
		reply, err := redis.Int(stub.Do("EXPIRE", key, expire))
		if err != nil {
			return nil, err
		}
		if reply != 1 {
			stub.Do("HSET", key, emptyHashField, emptyHashValue)
			stub.Do("EXPIRE", key, nilCacheExpire)
		}
	}

	return loaded, nil
}
