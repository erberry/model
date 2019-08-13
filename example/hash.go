package main

import (
	"context"
	"fmt"

	"github.com/erberry/model"
)

//将数据存储到redis的hash中

func hsimple() {
	db.DropTable(&ModelB{})

	if db.CreateTable(&ModelB{}).Error != nil {
		fmt.Println("failed")
		return
	}

	//insert
	ta := ModelB{Name: "hello", Age: 1}
	if err := db.Create(&ta).Error; err != nil {
		fmt.Println("failed")
		return
	}

	//select
	ta1 := ModelB{ID: ta.ID}
	if notFound, err := model.HMGet(context.TODO(), &ta1, nil, true); notFound || err != nil || ta1.Name != "hello" {
		fmt.Println("failed")
		return
	}

	//update all field
	ta1.Name = "world"
	ta1.Age = 0
	if model.HUpdate(context.TODO(), &ta1, nil) != nil {
		fmt.Println("failed")
		return
	}

	ta1.Name = ""
	if notFound, err := model.HMGet(context.TODO(), &ta1, []string{"haha", "age", "name"}, true); notFound || err != nil || ta1.Name != "world" {
		fmt.Println("failed")
		return
	}

	ta2 := ModelB{ID: ta.ID}
	//update target field
	if model.HUpdate(context.TODO(), &ta2, map[string]interface{}{"name": "golang", "age": 10}) != nil {
		fmt.Println("failed")
		return
	}

	if notFound, err := model.HMGet(context.TODO(), &ta2, nil, true); notFound || err != nil ||
		ta2.Name != "golang" || ta2.Age != 10 {
		fmt.Println("failed")
		return
	}

	model.HIncr(context.TODO(), &ta2, map[string]interface{}{"age": -100, "score": 20})

	t3 := ModelB{Name: "t3", Age: 1}
	if err := db.Create(&t3).Error; err != nil {
		fmt.Println("failed")
		return
	}
	rs, err := model.BatchHMGet(context.TODO(), ModelB{}, []interface{}{1, 2, 3, 4}, []string{"name", "aa", "age"}, true)
	if err != nil {
		fmt.Println("failed")
		return
	}
	if ts, ok := rs.([]ModelB); !ok || len(ts) != 2 || ts[0].Name != "golang" ||
		ts[1].Name != "t3" {
		fmt.Println("failed")
	} else {
		fmt.Println(ts)
		fmt.Println(ts[0])
		fmt.Println(ts[1])
	}

	model.HMGet(context.TODO(), &ModelB{ID: 3}, nil, true)
	fmt.Println("hash success")
}
