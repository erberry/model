package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/erberry/model"
)

func (t ModelA) RedisListKey() string {
	return "table_20190719:list"
}

func (t ModelA) Load(offset, limit int) ([]model.Z, error) {
	var arr []ModelA
	err := db.Select("id, age").Order("age").Offset(offset).Limit(limit).Find(&arr).Error
	if err != nil {
		return nil, err
	}
	zs := make([]model.Z, 0, len(arr))
	for _, a := range arr {
		zs = append(zs, model.Z{
			Field: strconv.Itoa(a.ID),
			Score: strconv.Itoa(a.Age),
		})
	}
	return zs, nil
}

func (t ModelA) GetLocker() model.Locker {
	locker := model.NewLocker(etcdClient, t.RedisListKey()+":lock", 3*time.Second)
	return &locker
}

func list() {
	db.DropTable(&ModelA{})

	if db.CreateTable(&ModelA{}).Error != nil {
		fmt.Println("failed")
		return
	}

	for i := 0; i < 5500; i++ {
		//insert
		ta := ModelA{Name: "hello" + strconv.Itoa(i), Age: i}
		if model.Insert(context.TODO(), &ta) != nil || ta.ID == 0 {
			fmt.Println("failed")
			return
		}
	}

	a := ModelA{}
	var offset, limit int
	limit = 50
	for i := 0; i < 120; i++ {
		offset = i * limit
		zs, total, err := model.GetByPage(a, offset, limit, false)
		if err != nil {
			fmt.Println("failed")
			return
		}
		if total != 5500 {
			fmt.Println("failed")
			return
		}

		for j, z := range zs {
			if id := z.Field; id != strconv.Itoa(i*limit+j+1) {
				fmt.Println("failed")
				return
			}
			if z.Score != strconv.Itoa(i*limit+j) {
				fmt.Println("failed")
				return
			}
		}
	}

	fmt.Println("success")
}
