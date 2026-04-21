package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"datacenter/internal/models"
	"datacenter/internal/storage"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var modules = []string{"movie", "music", "book", "game", "product"}

var movieTitles = []string{
	"Harry Potter and the Sorcerer's Stone", "The Lord of the Rings: The Fellowship of the Ring",
	"Star Wars: Episode IV - A New Hope", "Avengers: Endgame", "Spider-Man: Homecoming",
	"The Dark Knight", "Inception", "Interstellar", "The Matrix", "Fight Club",
}

var musicTitles = []string{
	"Abbey Road", "Revolver", "The Dark Side of the Moon", "Hotel California",
	"Back in Black", "Nevermind", "Thriller", "The Wall", "Led Zeppelin IV", "Rumours",
}

var bookTitles = []string{
	"To Kill a Mockingbird", "Pride and Prejudice", "1984", "Animal Farm",
	"The Great Gatsby", "Brave New World", "The Lord of the Flies", "Jane Eyre",
	"The Da Vinci Code", "Gone Girl",
}

var gameTitles = []string{
	"The Legend of Zelda: Breath of the Wild", "Super Mario Odyssey",
	"Red Dead Redemption 2", "The Last of Us", "God of War", "Elden Ring",
	"Spider-Man PS4", "Horizon Zero Dawn", "GTA V", "Monster Hunter World",
}

var productNames = []string{
	"iPhone 14 Pro", "MacBook Pro 16-inch", "iPad Pro 12.9-inch",
	"Apple Watch Series 8", "AirPods Pro", "Samsung Galaxy S23 Ultra",
	"Sony PlayStation 5", "Nintendo Switch OLED", "Dell XPS 15", "LG C2 65-inch TV",
}

func main() {
	fmt.Println("正在连接到业务数据库...")
	store, err := storage.NewMongoDBStorage(
		"mongodb://localhost:27017",
		"datacenter",
	)
	if err != nil {
		panic("连接数据库失败: " + err.Error())
	}

	ctx := context.Background()

	fmt.Println("正在清理所有集合...")
	collectionsToClean := []string{
		"collections", "custom_fields", "scrape_tasks", "deleted_scrape_tasks",
		"users", "roles", "permissions",
	}

	dynamicCollections := []string{
		"movie_data", "book_data", "music_data", "game_data", "product_data",
	}

	for _, collName := range collectionsToClean {
		err := store.GetDynamicCollection(collName).Drop(ctx)
		if err != nil {
			fmt.Printf("  删除集合 %s: %v\n", collName, err)
		} else {
			fmt.Printf("  删除集合 %s: 成功\n", collName)
		}
	}

	for _, collName := range dynamicCollections {
		err := store.GetDynamicCollection(collName).Drop(ctx)
		if err != nil {
			fmt.Printf("  删除集合 %s: %v\n", collName, err)
		} else {
			fmt.Printf("  删除集合 %s: 成功\n", collName)
		}
	}

	fmt.Println("\n正在创建模块集合...")
	rand.Seed(time.Now().UnixNano())

	for _, module := range modules {
		collection := &models.Collection{
			Module:         module,
			Description:    fmt.Sprintf("%s数据模块", module),
			DatatypeOwner:  "admin",
			CollectionName: module + "_data",
			ID:             primitive.NewObjectID(),
			BaseModel: models.BaseModel{
				CreatedBy: "admin",
				CreatedAt: time.Now(),
				UpdatedBy: "admin",
				UpdatedAt: time.Now(),
			},
		}
		err = store.CreateCollection(collection)
		if err != nil {
			fmt.Printf("创建模块集合 %s 失败: %v\n", module, err)
		} else {
			fmt.Printf("创建模块集合: %s 成功\n", module)
		}
	}

	fmt.Println("\n正在创建自定义字段...")
	createFieldDefinitions(ctx, store)

	fmt.Println("\n正在创建业务数据...")
	createBusinessData(ctx, store)

	fmt.Println("\n正在创建刮削任务...")
	createScrapeTasks(ctx, store)

	fmt.Println("\n所有数据创建完成!")
}

func createFieldDefinitions(ctx context.Context, store storage.Storage) {
	fieldDefs := map[string][]struct {
		fieldName string
		fieldType string
		required  bool
	}{
		"movie": {
			{"title", "string", true},
			{"director", "string", false},
			{"year", "number", false},
			{"rating", "number", false},
			{"genre", "string", false},
			{"duration", "number", false},
		},
		"book": {
			{"title", "string", true},
			{"author", "string", false},
			{"publisher", "string", false},
			{"year", "number", false},
			{"pages", "number", false},
		},
		"music": {
			{"title", "string", true},
			{"artist", "string", false},
			{"album", "string", false},
			{"year", "number", false},
			{"duration", "number", false},
		},
		"game": {
			{"title", "string", true},
			{"developer", "string", false},
			{"publisher", "string", false},
			{"year", "number", false},
			{"platform", "string", false},
		},
		"product": {
			{"name", "string", true},
			{"brand", "string", false},
			{"category", "string", false},
			{"price", "number", false},
		},
	}

	for module, fields := range fieldDefs {
		for _, field := range fields {
			fieldType := models.FieldTypeString
			if field.fieldType == "number" {
				fieldType = models.FieldTypeNumber
			} else if field.fieldType == "boolean" {
				fieldType = models.FieldTypeBoolean
			} else if field.fieldType == "date" {
				fieldType = models.FieldTypeDate
			}
			fieldDef := &models.FieldDefinition{
				Module:      module,
				FieldName:   field.fieldName,
				FieldType:   fieldType,
				Description: fmt.Sprintf("%s field", field.fieldName),
				Required:    field.required,
				ID:          primitive.NewObjectID(),
				BaseModel: models.BaseModel{
					CreatedBy: "admin",
					CreatedAt: time.Now(),
					UpdatedBy: "admin",
					UpdatedAt: time.Now(),
				},
			}
			err := store.CreateFieldDefinition(fieldDef)
			if err != nil {
				fmt.Printf("  创建字段 %s.%s 失败: %v\n", module, field.fieldName, err)
			}
		}
		fmt.Printf("  模块 %s 字段创建完成\n", module)
	}
}

func createBusinessData(ctx context.Context, store storage.Storage) {
	dataCount := 0

	for _, module := range modules {
		count := 10
		for i := 0; i < count; i++ {
			data := generateBusinessData(ctx, store, module, i)
			dataCount++
			fmt.Printf("  创建数据: %s - %s\n", module, data.Description)
		}
	}

	fmt.Printf("共创建 %d 条业务数据\n", dataCount)
}

func generateBusinessData(ctx context.Context, store storage.Storage, module string, index int) *models.BusinessData {
	var title string
	customFields := make(map[string]interface{})

	switch module {
	case "movie":
		title = movieTitles[index%len(movieTitles)]
		customFields["director"] = fmt.Sprintf("Director %d", index+1)
		customFields["year"] = 2000 + index
		customFields["rating"] = float64(70+index%30) / 10.0
		customFields["genre"] = []string{"Action", "Adventure", "Fantasy", "Drama"}[index%4]
		customFields["duration"] = 120 + index*5
	case "book":
		title = bookTitles[index%len(bookTitles)]
		customFields["author"] = fmt.Sprintf("Author %d", index+1)
		customFields["publisher"] = fmt.Sprintf("Publisher %d", index+1)
		customFields["year"] = 2000 + index
		customFields["pages"] = 200 + index*20
	case "music":
		title = musicTitles[index%len(musicTitles)]
		customFields["artist"] = fmt.Sprintf("Artist %d", index+1)
		customFields["album"] = fmt.Sprintf("Album %d", index+1)
		customFields["year"] = 2000 + index
		customFields["duration"] = 180 + index*10
	case "game":
		title = gameTitles[index%len(gameTitles)]
		customFields["developer"] = fmt.Sprintf("Developer %d", index+1)
		customFields["publisher"] = fmt.Sprintf("Publisher %d", index+1)
		customFields["year"] = 2000 + index
		customFields["platform"] = []string{"PC", "PS5", "Xbox", "Nintendo"}[index%4]
	case "product":
		title = productNames[index%len(productNames)]
		customFields["brand"] = fmt.Sprintf("Brand %d", index+1)
		customFields["category"] = []string{"Electronics", "Computers", "Audio", "Wearables"}[index%4]
		customFields["price"] = float64(1000 + index*100)
		customFields["stock"] = 50 + index*10
	}

	customFields["title"] = title

	data := &models.BusinessData{
		Module:       module,
		Description:  fmt.Sprintf("%s - %s", module, title),
		CustomFields: customFields,
		FilePath:     fmt.Sprintf("/data/%s/%d.json", module, index+1),
		ID:           primitive.NewObjectID(),
		BaseModel: models.BaseModel{
			CreatedBy: "admin",
			CreatedAt: time.Now(),
			UpdatedBy: "admin",
			UpdatedAt: time.Now(),
		},
	}

	err := store.CreateBusinessData(ctx, module+"_data", data)
	if err != nil {
		fmt.Printf("Error creating business data: %v\n", err)
	}

	return data
}

func createScrapeTasks(ctx context.Context, store storage.Storage) {
	taskCount := 0

	for i := 0; i < 20; i++ {
		module := modules[i%len(modules)]
		taskID := primitive.NewObjectID()
		createdAt := time.Now().Add(-time.Duration(i) * time.Hour)

		statusRoll := i % 5
		var status models.ScrapeTaskStatus
		var startedAt, completedAt *time.Time
		var errorMessage string

		if statusRoll == 0 {
			status = models.ScrapeTaskStatusPending
		} else if statusRoll == 1 {
			status = models.ScrapeTaskStatusScraping
			start := createdAt.Add(1 * time.Minute)
			startedAt = &start
		} else if statusRoll == 2 || statusRoll == 3 {
			status = models.ScrapeTaskStatusSuccess
			start := createdAt.Add(1 * time.Minute)
			startedAt = &start
			completed := start.Add(5 * time.Second)
			completedAt = &completed
		} else {
			status = models.ScrapeTaskStatusFailed
			start := createdAt.Add(1 * time.Minute)
			startedAt = &start
			completed := start.Add(2 * time.Second)
			completedAt = &completed
			errorMessages := []string{
				"刮削器执行失败: Python脚本语法错误",
				"数据文件不存在: 文件已被移动或删除",
				"刮削器执行失败: 超时",
			}
			errorMessage = errorMessages[i%len(errorMessages)]
		}

		task := &models.ScrapeTask{
			ID: taskID,
			BaseModel: models.BaseModel{
				CreatedBy: "admin",
				CreatedAt: createdAt,
				UpdatedBy: "admin",
				UpdatedAt: createdAt,
			},
			Module:         module,
			DataPath:       fmt.Sprintf("/data/%s/%d.json", module, i+1),
			ScraperPath:    fmt.Sprintf("/scrapers/%s_scraper.py", module),
			Status:         status,
			Result:         map[string]interface{}{"items_scraped": rand.Intn(100) + 1},
			ErrorMessage:   errorMessage,
			StartedAt:      startedAt,
			CompletedAt:    completedAt,
			Description:    fmt.Sprintf("刮削任务 %d", i+1),
		}

		err := store.CreateScrapeTask(task)
		if err != nil {
			fmt.Printf("  创建刮削任务失败: %v\n", err)
		} else {
			taskCount++
			fmt.Printf("  创建刮削任务: %s - %s\n", module, status)
		}
	}

	fmt.Printf("共创建 %d 个刮削任务\n", taskCount)
}
