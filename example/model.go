package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/erberry/model"
	"github.com/jinzhu/gorm"
)

//ModelA ModelA
type ModelA struct {
	ID   int    `redis:"id"`
	Name string `redis:"name"`
	Age  int    `redis:"age"`
}

func (t ModelA) Where(ids interface{}) *gorm.DB {
	if ids != nil {
		return db.Where("id in (?)", ids)
	}
	return db.Where("id = ?", t.ID)
}

func (t ModelA) TableName() string {
	return "table_20190719"
}

func (t ModelA) RedisKey(id interface{}) string {
	if t.ID != 0 {
		return fmt.Sprintf("string:%s", strconv.Itoa(t.ID))
	}

	return fmt.Sprintf("string:%s", strconv.Itoa(id.(int)))
}
func (t ModelA) RedisStub() model.RedisStub {
	return pool.Get()
}
func (t ModelA) RedisExpire() time.Duration {
	return 200 * time.Second
}

//ModelB ModelB
type ModelB struct {
	ID   int    `redis:"id"`
	Name string `redis:"name"`
	Age  int    `redis:"age"`
}

func (t ModelB) RedisKey(id interface{}) string {
	if t.ID != 0 {
		return fmt.Sprintf("hash:%s", strconv.Itoa(t.ID))
	}

	return fmt.Sprintf("hash:%s", strconv.Itoa(id.(int)))
}

func (t ModelB) Where(ids interface{}) *gorm.DB {
	if ids != nil {
		return db.Where("id in (?)", ids)
	}
	return db.Where("id = ?", t.ID)
}

func (t ModelB) TableName() string {
	return "table_20190719"
}

func (t ModelB) RedisStub() model.RedisStub {
	return pool.Get()
}
func (t ModelB) RedisExpire() time.Duration {
	return 200 * time.Second
}
