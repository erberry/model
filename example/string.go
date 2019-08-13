package main

import (
	"context"
	"fmt"

	"github.com/erberry/model"
)

//将数据转为json串，存储到redis的string类型中

func simple() {
	db.DropTable(&ModelA{})

	if db.CreateTable(&ModelA{}).Error != nil {
		fmt.Println("failed")
		return
	}

	//insert
	ta := ModelA{Name: "hello", Age: 1}
	if err := db.Create(&ta).Error; err != nil {
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

	if err := db.Create(&ta2).Error; err != nil {
		fmt.Println("failed")
		return
	}
	if err := db.Create(&ta3).Error; err != nil {
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

	fmt.Println("string success")
}
