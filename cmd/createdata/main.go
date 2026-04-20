package main

import (
	"context"
	"fmt"
	"time"

	"datacenter/internal/models"
	"datacenter/internal/storage"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	fmt.Println("正在连接到业务数据库...")
	storage, err := storage.NewMongoDBStorage(
		"mongodb://localhost:27017",
		"datacenter",
	)
	if err != nil {
		panic("连接数据库失败: " + err.Error())
	}

	// 确保测试模块集合存在
	module := "test_upload"
	_, err = storage.GetCollectionByModule(module)
	if err != nil {
		collection := &models.Collection{
			Module:         module,
			Description:    "测试上传模块",
			DatatypeOwner:  "test",
			CollectionName: module + "_data",
			ID:             primitive.NewObjectID(),
			BaseModel: models.BaseModel{
				CreatedBy: "test",
				CreatedAt: time.Now(),
				UpdatedBy: "test",
				UpdatedAt: time.Now(),
			},
		}
		err = storage.CreateCollection(collection)
		if err != nil {
			fmt.Printf("创建测试模块集合失败: %v\n", err)
			return
		}
		fmt.Printf("创建测试模块集合: %s 成功\n", module)
	} else {
		fmt.Printf("测试模块集合 %s 已存在\n", module)
	}

	// 创建测试数据
	for i := 0; i < 15; i++ {
		data := &models.BusinessData{
			ID: primitive.NewObjectID(),
			BaseModel: models.BaseModel{
				CreatedBy: "test",
				CreatedAt: time.Now(),
				UpdatedBy: "test",
				UpdatedAt: time.Now(),
			},
			Module:      module,
			Description: fmt.Sprintf("测试数据 %d", i+1),
			CustomFields: map[string]interface{}{
				"title":    fmt.Sprintf("测试标题 %d", i+1),
				"value":    i + 1,
				"category": "测试类别",
			},
			FilePath: fmt.Sprintf("/data/test_%d.json", i+1),
		}

		err := storage.CreateBusinessData(context.Background(), data)
		if err != nil {
			fmt.Printf("创建测试数据失败: %v\n", err)
			continue
		}
		fmt.Printf("创建测试数据: %s\n", data.Description)
	}

	fmt.Println("测试数据创建完成!")

	// 测试获取数据总量
	total, err := storage.GetBusinessDataCount(module, map[string]interface{}{})
	if err != nil {
		fmt.Printf("获取数据总量失败: %v\n", err)
		return
	}
	fmt.Printf("数据总量: %d\n", total)
}
