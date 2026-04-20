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
	fmt.Println("开始创建测试数据...")

	// 连接业务数据库
	businessStorage, err := storage.NewMongoDBStorage(
		"mongodb://localhost:27017",
		"datacenter",
	)
	if err != nil {
		panic("连接业务数据库失败: " + err.Error())
	}

	// 连接RBAC数据库
	rbacStorage, err := storage.NewMongoDBStorage(
		"mongodb://localhost:27017",
		"rbac",
	)
	if err != nil {
		panic("连接RBAC数据库失败: " + err.Error())
	}

	// 1. 创建权限
	createPermissions(rbacStorage)

	// 2. 创建角色
	createRoles(rbacStorage)

	// 3. 创建用户
	createUsers(rbacStorage)

	// 4. 创建模块集合
	createCollections(businessStorage)

	// 5. 创建刮削任务和业务数据
	createScrapeData(businessStorage)

	fmt.Println("测试数据创建完成！")
}

func createPermissions(s storage.Storage) {
	permissions := []models.Permission{
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "system", CreatedAt: time.Now(), UpdatedBy: "system", UpdatedAt: time.Now()}, Name: "用户管理", Code: "user:manage", Description: "管理系统用户账户"},
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "system", CreatedAt: time.Now(), UpdatedBy: "system", UpdatedAt: time.Now()}, Name: "角色管理", Code: "role:manage", Description: "管理系统角色"},
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "system", CreatedAt: time.Now(), UpdatedBy: "system", UpdatedAt: time.Now()}, Name: "权限管理", Code: "permission:manage", Description: "管理系统权限"},
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "system", CreatedAt: time.Now(), UpdatedBy: "system", UpdatedAt: time.Now()}, Name: "数据管理", Code: "data:manage", Description: "管理业务数据"},
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "system", CreatedAt: time.Now(), UpdatedBy: "system", UpdatedAt: time.Now()}, Name: "刮削管理", Code: "scraper:manage", Description: "管理刮削任务"},
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "system", CreatedAt: time.Now(), UpdatedBy: "system", UpdatedAt: time.Now()}, Name: "集合管理", Code: "collection:manage", Description: "管理数据集合"},
	}

	for _, perm := range permissions {
		err := s.CreatePermission(&perm)
		if err != nil {
			fmt.Printf("创建权限 %s 失败: %v\n", perm.Name, err)
		} else {
			fmt.Printf("创建权限: %s\n", perm.Name)
		}
	}
}

func createRoles(s storage.Storage) {
	// 先获取所有权限
	permissions, err := s.GetPermissions(0, 100)
	if err != nil {
		fmt.Printf("获取权限失败: %v\n", err)
		return
	}

	// 收集权限ID
	permissionIDs := make([]string, len(permissions))
	for i, perm := range permissions {
		permissionIDs[i] = perm.ID.Hex()
	}

	roles := []models.Role{
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "system", CreatedAt: time.Now(), UpdatedBy: "system", UpdatedAt: time.Now()}, Name: "超级管理员", Code: "admin", Description: "系统超级管理员", PermissionIDs: permissionIDs},
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "system", CreatedAt: time.Now(), UpdatedBy: "system", UpdatedAt: time.Now()}, Name: "数据管理员", Code: "data_admin", Description: "数据管理权限", PermissionIDs: permissionIDs[3:6]},
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "system", CreatedAt: time.Now(), UpdatedBy: "system", UpdatedAt: time.Now()}, Name: "普通用户", Code: "user", Description: "普通用户权限", PermissionIDs: permissionIDs[3:4]},
	}

	for _, role := range roles {
		err := s.CreateRole(&role)
		if err != nil {
			fmt.Printf("创建角色 %s 失败: %v\n", role.Name, err)
		} else {
			fmt.Printf("创建角色: %s\n", role.Name)
		}
	}
}

func createUsers(s storage.Storage) {
	// 先获取所有角色
	roles, err := s.GetRoles(0, 100)
	if err != nil {
		fmt.Printf("获取角色失败: %v\n", err)
		return
	}

	// 收集角色ID
	roleIDs := make([]string, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID.Hex()
	}

	users := []models.User{
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "system", CreatedAt: time.Now(), UpdatedBy: "system", UpdatedAt: time.Now()}, Username: "admin", Password: "liangminchuan", Email: "admin@datacenter.local", RoleIDs: roleIDs},
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "admin", CreatedAt: time.Now(), UpdatedBy: "admin", UpdatedAt: time.Now()}, Username: "dataadmin", Password: "liangminchuan", Email: "dataadmin@datacenter.local", RoleIDs: roleIDs[1:2]},
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "admin", CreatedAt: time.Now(), UpdatedBy: "admin", UpdatedAt: time.Now()}, Username: "user1", Password: "liangminchuan", Email: "user1@datacenter.local", RoleIDs: roleIDs[2:3]},
		{ID: primitive.NewObjectID(), BaseModel: models.BaseModel{CreatedBy: "admin", CreatedAt: time.Now(), UpdatedBy: "admin", UpdatedAt: time.Now()}, Username: "user2", Password: "liangminchuan", Email: "user2@datacenter.local", RoleIDs: roleIDs[2:3]},
	}

	for _, user := range users {
		err := s.CreateUser(&user)
		if err != nil {
			fmt.Printf("创建用户 %s 失败: %v\n", user.Username, err)
		} else {
			fmt.Printf("创建用户: %s (密码: liangminchuan)\n", user.Username)
		}
	}
}

func createCollections(s storage.Storage) {
	modules := []string{"movie", "music", "book", "game", "product"}
	descriptions := map[string]string{
		"movie":   "电影数据模块",
		"music":   "音乐数据模块",
		"book":    "图书数据模块",
		"game":    "游戏数据模块",
		"product": "产品数据模块",
	}

	for _, module := range modules {
		collection := &models.Collection{
			ID: primitive.NewObjectID(),
			BaseModel: models.BaseModel{
				CreatedBy: "admin",
				CreatedAt: time.Now(),
				UpdatedBy: "admin",
				UpdatedAt: time.Now(),
			},
			Module:         module,
			Description:    descriptions[module],
			DatatypeOwner:  "admin",
			CollectionName: module + "_data",
		}

		err := s.CreateCollection(collection)
		if err != nil {
			fmt.Printf("创建模块 %s 失败: %v\n", module, err)
		} else {
			fmt.Printf("创建模块: %s\n", module)
		}
	}
}

func createScrapeData(s storage.Storage) {
	modules := []string{"movie", "music", "book", "game", "product"}
	scraperPaths := map[string]string{
		"movie":   "/scrapers/movie_scraper.py",
		"music":   "/scrapers/music_scraper.py",
		"book":    "/scrapers/book_scraper.py",
		"game":    "/scrapers/game_scraper.py",
		"product": "/scrapers/product_scraper.py",
	}

	// 为每个模块创建20个刮削任务
	for _, module := range modules {
		for i := 0; i < 20; i++ {
			// 创建刮削任务
			task := &models.ScrapeTask{
				ID: primitive.NewObjectID(),
				BaseModel: models.BaseModel{
					CreatedBy: "admin",
					CreatedAt: time.Now(),
					UpdatedBy: "admin",
					UpdatedAt: time.Now(),
				},
				Module:      module,
				DataPath:    fmt.Sprintf("/data/%s/data_%d.json", module, i+1),
				ScraperPath: scraperPaths[module],
				Status:      models.ScrapeTaskStatusSuccess,
				Result: map[string]interface{}{
					"items_scraped": 10,
					"duration_ms":   1500,
				},
				ErrorMessage: "",
			}

			// 模拟时间
			startedAt := time.Now().Add(-time.Hour)
			completedAt := time.Now().Add(-time.Hour + time.Second*30)
			task.StartedAt = &startedAt
			task.CompletedAt = &completedAt

			// 创建刮削任务
			err := s.CreateScrapeTask(task)
			if err != nil {
				fmt.Printf("创建刮削任务失败: %v\n", err)
				continue
			}

			// 创建对应的业务数据
			data := &models.BusinessData{
				ID: primitive.NewObjectID(),
				BaseModel: models.BaseModel{
					CreatedBy: "admin",
					CreatedAt: time.Now(),
					UpdatedBy: "admin",
					UpdatedAt: time.Now(),
				},
				Module:      module,
				Description: fmt.Sprintf("%s 刮削数据 %d", module, i+1),
				CustomFields: map[string]interface{}{
					"title":       fmt.Sprintf("%s Item %d", module, i+1),
					"scrape_path": scraperPaths[module],
					"data_path":   task.DataPath,
					"task_id":     task.ID.Hex(),
					"scraped_at":  completedAt,
				},
				FilePath: task.DataPath,
			}

			err = s.CreateBusinessData(context.Background(), data)
			if err != nil {
				fmt.Printf("创建业务数据失败: %v\n", err)
				continue
			}

			// 更新刮削任务的业务数据ID
			task.BusinessDataID = data.ID
			err = s.UpdateScrapeTask(task)
			if err != nil {
				fmt.Printf("更新刮削任务失败: %v\n", err)
			}

			if i%5 == 0 {
				fmt.Printf("已创建 %s 模块 %d 个刮削任务\n", module, i+1)
			}
		}
	}

	fmt.Println("刮削任务和业务数据创建完成！")
}
