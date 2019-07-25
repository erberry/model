package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/erberry/model"
	"github.com/jinzhu/gorm"
)

//ModelA ModelA
type ModelA struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (t ModelA) Slave() *gorm.DB {
	return db
}

func (t ModelA) Master() *gorm.DB {
	return db
}

func (t ModelA) Where() *gorm.DB {
	return nil
	//return t.Slave().Where("name = ?", "world")
}

func (t ModelA) TableName() string {
	return "table_20190719"
}

func (t ModelA) RedisKey(id interface{}) string {
	if t.ID != 0 {
		return fmt.Sprintf("table_20190719:%s", strconv.Itoa(t.ID))
	}

	return fmt.Sprintf("table_20190719:%s", strconv.Itoa(id.(int)))
}
func (t ModelA) RedisStub() model.RedisStub {
	return pool.Get()
}
func (t ModelA) Expire() int {
	return 20
}

func simple() {
	db.DropTable(&ModelA{})

	if db.CreateTable(&ModelA{}).Error != nil {
		fmt.Println("failed")
		return
	}

	//insert
	ta := ModelA{Name: "hello", Age: 1}
	if model.Insert(context.TODO(), &ta) != nil || ta.ID == 0 {
		fmt.Println("failed")
		return
	}

	//select
	ta1 := ModelA{ID: ta.ID}
	if notFound, err := model.Get(context.TODO(), &ta1, true); notFound || err != nil || ta1.Name != "hello" {
		fmt.Println("failed")
		return
	}

	//update all field
	ta1.Name = "world"
	ta1.Age = 0
	if model.Update(context.TODO(), &ta1, nil) != nil {
		fmt.Println("failed")
		return
	}

	ta1.Name = ""
	if notFound, err := model.Get(context.TODO(), &ta1, true); notFound || err != nil || ta1.Name != "world" {
		fmt.Println("failed")
		return
	}

	//update target field
	if model.Update(context.TODO(), &ta1, map[string]interface{}{"age": 20}) != nil {
		fmt.Printf("failed")
		return
	}

	ta1.Age = 0
	if notFound, err := model.Get(context.TODO(), &ta1, true); notFound || err != nil || ta1.Age != 20 {
		fmt.Println("failed")
		return
	}

	//delete
	if model.Delete(context.TODO(), &ta1) != nil {
		fmt.Println("failed")
		return
	}

	ta2 := ModelA{
		Name: "m2",
		Age:  10,
	}
	ta3 := ModelA{
		Name: "m3",
		Age:  11,
	}

	if model.Insert(context.TODO(), &ta2) != nil {
		fmt.Println("failed")
		return
	}
	if model.Insert(context.TODO(), &ta3) != nil {
		fmt.Println("failed")
		return
	}

	ret, err := model.BatchGet(context.TODO(), ModelA{}, []int{ta1.ID, ta2.ID, ta3.ID, 1001, 1002}, true)
	if err != nil {
		fmt.Println("failed")
		return
	}
	if ta, ok := ret.([]ModelA); !ok || ta[0].Name != "m2" || ta[1].Name != "m3" {
		fmt.Println("failed")
		return
	}

	fmt.Println("success")
}
