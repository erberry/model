package model

import (
	"context"
	"reflect"
	"time"

	goRedis "github.com/go-redis/redis"
	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
	jsoniter "github.com/json-iterator/go"
)

//Model Model
type Model interface {
	//查询条件 批量查询传入ids
	Where(ids interface{}) *gorm.DB
	//表名称
	TableName() string

	//RedisKey
	RedisKey(id interface{}) string
	//RedisStub
	RedisStub() RedisStub
	//redis 过期时间 秒
	RedisExpire()  time.Duration
}

const (
	nilCacheExpire     = 1                        //避免缓存击穿，空记录在redis中的过期时间(秒)
	emptyRecordContent = "empty-5030573512345671" //空记录在redis中的内容
	emptyHashField     = "empty-5030573512345671"
	emptyHashValue     = "empty-v-5030573512345671"
	mustExistField     = "must-exist-5030573512345671"
	mustExistValue     = "must-exist-v-5030573512345671"
	defaultCacheExpire = 60 * 60
)

//Get Where接口返回的条件查询
func Get(ctx context.Context, m Model, fromCache bool) (notFound bool, err error) {
	if !fromCache {
		return load(m, nil)
	}

	stub := m.RedisStub()
	key := m.RedisKey(nil)

	data, err := redis.String(stub.Do("GET", key))
	if err == redis.ErrNil || err == goRedis.Nil {
		return flushCache(ctx, m)
	}

	if err != nil {
		return false, err
	}

	if len(data) == 0 {
		return true, nil
	}

	err = jsoniter.Unmarshal([]byte(data), m)
	if err != nil {
		return false, err
	}

	return false, nil
}

//BatchGet Where接口返回的条件查询
func BatchGet(ctx context.Context, m Model, ids interface{}, fromCache bool) (interface{}, error) {
	if !fromCache {
		return batchLoad(m, ids)
	}

	stub := m.RedisStub()
	v := reflect.ValueOf(ids)

	keys := make([]interface{}, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		key := m.RedisKey(v.Index(i).Interface())
		keys = append(keys, key)
	}

	data, err := redis.Strings(stub.Do("MGET", keys...))
	if err != nil {
		return nil, err
	}

	noCached := reflect.MakeSlice(reflect.TypeOf(ids), 0, 0)
	elemType := reflect.ValueOf(m).Type()
	cachedResults := reflect.MakeSlice(reflect.SliceOf(elemType), 0, 0)

	for i := 0; i < len(data); i++ {
		if data[i] != emptyRecordContent {
			elemType := reflect.ValueOf(m).Type()
			elem := reflect.New(elemType)
			err = jsoniter.Unmarshal([]byte(data[i]), elem.Interface())
			if err != nil {
				noCached = reflect.Append(noCached, v.Index(i))
			} else {
				cachedResults = reflect.Append(cachedResults, elem.Elem())
			}
		}
	}

	if noCached.Len() > 0 {
		noCachedResults, err := multiFlushCache(ctx, m, noCached.Interface())
		if err != nil {
			return cachedResults.Interface(), err
		}

		return reflect.AppendSlice(cachedResults, reflect.ValueOf(noCachedResults)).Interface(), nil
	}

	return cachedResults.Interface(), nil
}

//Delete Where接口返回的条件Delete
func Delete(ctx context.Context, m Model) error {
	err := delete(m)
	stub := m.RedisStub()
	if stub != nil && err == nil {
		stub.Do("DEL", m.RedisKey(nil))
	}
	return err
}

//Update Where接口返回的条件Update。如果fields为空，更新所有字段
func Update(ctx context.Context, m Model, fields map[string]interface{}) error {
	err := update(m, fields)
	stub := m.RedisStub()
	if stub != nil && err == nil {
		stub.Do("DEL", m.RedisKey(nil))
	}
	return err
}

func flushCache(ctx context.Context, m Model) (bool, error) {
	noRecord, err := load(m, nil)
	if err != nil {
		return false, err
	}

	stub := m.RedisStub()
	key := m.RedisKey(nil)

	if noRecord {
		stub.Do("SET", key, "")
	} else {
		s, err := jsoniter.Marshal(m)
		if err != nil {
			return false, err
		}
		stub.Do("SET", key, string(s))
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

func multiFlushCache(ctx context.Context, m Model, ids interface{}) (interface{}, error) {
	loaded, err := batchLoad(m, ids)
	if err != nil {
		return nil, err
	}

	stub := m.RedisStub()

	var cmds []interface{}
	v := reflect.ValueOf(loaded)
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface().(Model)
		s, err := jsoniter.Marshal(item)
		if err != nil {
			return nil, err
		}
		key := item.RedisKey(nil)
		cmds = append(cmds, key, s)
	}

	if len(cmds) > 0 {
		stub.Do("MSET", cmds...)
	}

	expire := int(m.RedisExpire().Seconds())
	if expire <= 0 {
		expire = defaultCacheExpire
	}

	v = reflect.ValueOf(ids)
	for i := 0; i < v.Len(); i++ {
		key := m.RedisKey(v.Index(i).Interface())
		reply, err := redis.Int(stub.Do("EXPIRE", key, expire))
		if err != nil {
			return nil, err
		}
		if reply != 1 {
			stub.Do("SET", key, emptyRecordContent, "EX", nilCacheExpire, "NX")
		}
	}

	return loaded, nil
}

func delete(m Model) error {
	err := m.Where(nil).Delete(m).Error
	if err != nil {
		return err
	}
	return nil
}

func load(m Model, fields []string) (bool, error) {
	db := m.Where(nil)

	if fields != nil && len(fields) > 0 {
		db = db.Select(fields)
	}

	db = db.Take(m)
	if db.RecordNotFound() {
		return true, nil
	}
	if db.Error != nil {
		return false, db.Error
	}
	return false, nil
}

func update(m Model, fields map[string]interface{}) error {
	db := m.Where(nil)

	var err error
	if len(fields) == 0 {
		err = db.Model(m).Save(m).Error
	} else {
		err = db.Model(m).Updates(fields).Error
	}

	if err != nil {
		return err
	}
	return nil
}

func batchLoad(m Model, ids interface{}) (interface{}, error) {
	elemType := reflect.ValueOf(m).Type()
	sliceValue := reflect.MakeSlice(reflect.SliceOf(elemType), 0, 0)
	results := reflect.New(sliceValue.Type())
	iresults := results.Interface()

	db := m.Where(ids)

	db = db.Table(m.TableName()).Find(iresults)
	return results.Elem().Interface(), db.Error
}
