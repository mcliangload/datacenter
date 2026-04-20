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
	"Harry Potter and the Sorcerer's Stone", "Harry Potter and the Chamber of Secrets",
	"Harry Potter and the Prisoner of Azkaban", "Harry Potter and the Goblet of Fire",
	"Harry Potter and the Order of the Phoenix", "Harry Potter and the Half-Blood Prince",
	"Harry Potter and the Deathly Hallows", "The Lord of the Rings: The Fellowship of the Ring",
	"The Lord of the Rings: The Two Towers", "The Lord of the Rings: The Return of the King",
	"The Hobbit: An Unexpected Journey", "The Hobbit: The Desolation of Smaug",
	"The Hobbit: The Battle of the Five Armies", "Star Wars: Episode IV - A New Hope",
	"Star Wars: Episode V: The Empire Strikes Back", "Star Wars: Episode VI: Return of the Jedi",
	"Star Wars: Episode I: The Phantom Menace", "Star Wars: Episode II: Attack of the Clones",
	"Star Wars: Episode III: Revenge of the Sith", "Star Wars: Episode VII: The Force Awakens",
	"Avengers: Endgame", "Avengers: Infinity War", "Iron Man", "Thor", "Captain America",
	"Spider-Man: Homecoming", "Spider-Man: Far From Home", "Black Panther", "Doctor Strange",
	"Guardians of the Galaxy", "Ant-Man", "Black Widow", "Shang-Chi", "Eternals",
}

var musicTitles = []string{
	"Abbey Road", "Revolver", "Sgt. Pepper's Lonely Hearts Club Band", "The Beatles White Album",
	"Let It Be", "Help!", "A Hard Day's Night", "Rubber Soul", "Rubber Soul",
	"The Dark Side of the Moon", "The Wall", "Wish You Were Here", "Animals",
	"Hotel California", "Life in the Fast Lane", "Desperado",
	"Back in Black", "Highway to Hell", "Thunderstruck", "Shoot to Thrill",
	"Nevermind", "Smells Like Teen Spirit", "Come as You Are", "Lithium",
	"Thriller", "Beat It", "Billie Jean", "Smooth Criminal",
}

var bookTitles = []string{
	"To Kill a Mockingbird", "Pride and Prejudice", "1984", "Animal Farm",
	"The Great Gatsby", "The Catcher in the Rye", "Brave New World",
	"The Lord of the Flies", "Jane Eyre", "Wuthering Heights",
	"The Chronicles of Narnia", "The Lion, the Witch and the Wardrobe",
	"Harry Potter and the Sorcerer's Stone", "The Hobbit",
	"The Da Vinci Code", "Angels & Demons", "Inferno", "Lost Symbol",
	"The Girl with the Dragon Tattoo", "The Girl Who Played with Fire",
	"The Girl Who Kicked the Hornets' Nest", "Gone Girl", "Sharp Objects",
}

var gameTitles = []string{
	"The Legend of Zelda: Breath of the Wild", "Super Mario Odyssey",
	"Super Mario Galaxy", "Super Mario Bros. U", "Mario Kart 8 Deluxe",
	"Red Dead Redemption 2", "Red Dead Redemption", "GTA V", "GTA IV",
	"The Last of Us", "The Last of Us Part II", "Uncharted 4",
	"God of War", "God of War Ragnarok", "Spider-Man PS4", "Spider-Man PS5",
	"Horizon Zero Dawn", "Horizon Forbidden West", "Ghost of Tsushima",
	"Elden Ring", "Dark Souls", "Dark Souls II", "Dark Souls III",
	"Bloodborne", "Sekiro", "Monster Hunter World", "Monster Hunter Rise",
}

var productNames = []string{
	"iPhone 14 Pro", "iPhone 14", "iPhone 13 Pro", "iPhone 13",
	"MacBook Pro 16-inch", "MacBook Pro 14-inch", "MacBook Air M2",
	"iPad Pro 12.9-inch", "iPad Pro 11-inch", "iPad Air",
	"Apple Watch Series 8", "Apple Watch Ultra", "Apple Watch SE",
	"AirPods Pro", "AirPods", "AirPods Max",
	"Samsung Galaxy S23 Ultra", "Samsung Galaxy S23", "Samsung Galaxy S22",
	"Sony PlayStation 5", "Sony PlayStation 4 Pro", "Nintendo Switch OLED",
	"Dell XPS 15", "Dell XPS 13", "MacBook Pro 13-inch",
	"LG C2 65-inch TV", "Samsung QN90B 55-inch TV", "Sony A80K 65-inch TV",
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

	// 确保所有模块集合存在
	for _, module := range modules {
		collection := &models.Collection{
			Module:         module,
			Description:    fmt.Sprintf("%s数据模块", module),
			DatatypeOwner:  "admin",
			CollectionName: module + "_data",
			ID:             primitive.NewObjectID(),
			BaseModel: models.BaseModel{
				CreatedBy: "test",
				CreatedAt: time.Now(),
				UpdatedBy: "test",
				UpdatedAt: time.Now(),
			},
		}
		_, err = store.GetCollectionByModule(module)
		if err != nil {
			err = store.CreateCollection(collection)
			if err != nil {
				fmt.Printf("创建模块集合 %s 失败: %v\n", module, err)
			} else {
				fmt.Printf("创建模块集合: %s 成功\n", module)
			}
		} else {
			fmt.Printf("模块集合 %s 已存在\n", module)
		}
	}

	// 生成刮削任务
	rand.Seed(time.Now().UnixNano())
	totalTasks := 120
	taskCount := 0
	scrapedDataCount := 0

	for i := 0; i < totalTasks; i++ {
		module := modules[rand.Intn(len(modules))]
		taskID := primitive.NewObjectID()
		createdAt := time.Now().Add(-time.Duration(rand.Intn(30)) * 24 * time.Hour)

		// 根据时间决定状态分布
		statusRoll := rand.Float64()
		var status models.ScrapeTaskStatus
		var startedAt, completedAt *time.Time
		var errorMessage string
		var businessDataID primitive.ObjectID

		if statusRoll < 0.05 {
			status = models.ScrapeTaskStatusPending
		} else if statusRoll < 0.1 {
			status = models.ScrapeTaskStatusScraping
			start := createdAt.Add(time.Duration(rand.Intn(5)) * time.Minute)
			startedAt = &start
		} else if statusRoll < 0.7 {
			status = models.ScrapeTaskStatusSuccess
			start := createdAt.Add(time.Duration(rand.Intn(5)) * time.Minute)
			startedAt = &start
			completed := start.Add(time.Duration(100+rand.Intn(5000)) * time.Millisecond)
			completedAt = &completed

			// 为成功的任务创建对应的业务数据
			dataID, err := createBusinessData(ctx, store, module, taskID.Hex(), createdAt)
			if err == nil {
				businessDataID = dataID
				scrapedDataCount++
			}
		} else {
			status = models.ScrapeTaskStatusFailed
			start := createdAt.Add(time.Duration(rand.Intn(5)) * time.Minute)
			startedAt = &start
			completed := start.Add(time.Duration(100+rand.Intn(2000)) * time.Millisecond)
			completedAt = &completed
			errorMessages := []string{
				"刮削器执行失败: Python脚本语法错误",
				"数据文件不存在: 文件已被移动或删除",
				"刮削器执行失败: 超时",
				"解析刮削器输出失败: JSON格式错误",
				"刮削器执行失败: 权限不足",
			}
			errorMessage = errorMessages[rand.Intn(len(errorMessages))]
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
			DataPath:       fmt.Sprintf("/data/%s/%s_%d.json", module, module, i+1),
			ScraperPath:    fmt.Sprintf("/scrapers/%s_scraper.py", module),
			Status:         status,
			Result:         map[string]interface{}{"items_scraped": rand.Intn(100) + 1},
			ErrorMessage:   errorMessage,
			StartedAt:      startedAt,
			CompletedAt:    completedAt,
			BusinessDataID: businessDataID,
		}

		err = store.CreateScrapeTask(task)
		if err != nil {
			fmt.Printf("创建刮削任务失败: %v\n", err)
		} else {
			taskCount++
			if taskCount%20 == 0 {
				fmt.Printf("已创建 %d 个刮削任务...\n", taskCount)
			}
		}
	}

	fmt.Printf("\n测试数据生成完成!\n")
	fmt.Printf("刮削任务总数: %d\n", taskCount)
	fmt.Printf("刮削后业务数据总数: %d\n", scrapedDataCount)
}

func createBusinessData(ctx context.Context, store storage.Storage, module, taskID string, scrapedAt time.Time) (primitive.ObjectID, error) {
	var title string
	switch module {
	case "movie":
		title = movieTitles[rand.Intn(len(movieTitles))]
	case "music":
		title = musicTitles[rand.Intn(len(musicTitles))]
	case "book":
		title = bookTitles[rand.Intn(len(bookTitles))]
	case "game":
		title = gameTitles[rand.Intn(len(gameTitles))]
	case "product":
		title = productNames[rand.Intn(len(productNames))]
	default:
		title = "Unknown"
	}

	customFields := map[string]interface{}{
		"title":       title,
		"scrape_path": fmt.Sprintf("/scrapers/%s_scraper.py", module),
		"data_path":   fmt.Sprintf("/data/%s/", module),
		"module":      module,
		"task_id":     taskID,
		"scraped_at":  scrapedAt,
	}

	// 添加模块特定的字段
	switch module {
	case "movie":
		customFields["director"] = fmt.Sprintf("Director %d", rand.Intn(100))
		customFields["year"] = 2000 + rand.Intn(24)
		customFields["rating"] = float64(rand.Intn(30)+50) / 10.0
		customFields["genre"] = []string{"Action", "Adventure", "Fantasy"}[rand.Intn(3)]
	case "music":
		customFields["artist"] = fmt.Sprintf("Artist %d", rand.Intn(100))
		customFields["album"] = fmt.Sprintf("Album %d", rand.Intn(100))
		customFields["year"] = 2000 + rand.Intn(24)
		customFields["duration"] = rand.Intn(300) + 120
	case "book":
		customFields["author"] = fmt.Sprintf("Author %d", rand.Intn(100))
		customFields["publisher"] = fmt.Sprintf("Publisher %d", rand.Intn(50))
		customFields["year"] = 2000 + rand.Intn(24)
		customFields["pages"] = 100 + rand.Intn(500)
	case "game":
		customFields["developer"] = fmt.Sprintf("Developer %d", rand.Intn(50))
		customFields["publisher"] = fmt.Sprintf("Publisher %d", rand.Intn(50))
		customFields["year"] = 2000 + rand.Intn(24)
		customFields["platform"] = []string{"PC", "PS5", "Xbox", "Nintendo"}[rand.Intn(4)]
	case "product":
		customFields["brand"] = fmt.Sprintf("Brand %d", rand.Intn(50))
		customFields["category"] = []string{"Electronics", "Computers", "Audio", "Wearables"}[rand.Intn(4)]
		customFields["price"] = float64(rand.Intn(100000)) / 100.0
	}

	data := &models.BusinessData{
		Module:       module,
		Description:  fmt.Sprintf("刮削数据 - %s", title),
		CustomFields: customFields,
		FilePath:     fmt.Sprintf("/data/%s/", module),
		ID:           primitive.NewObjectID(),
		BaseModel: models.BaseModel{
			CreatedBy: "scraper",
			CreatedAt: scrapedAt,
			UpdatedBy: "scraper",
			UpdatedAt: scrapedAt,
		},
	}

	err := store.CreateBusinessData(ctx, data)
	if err != nil {
		return primitive.NilObjectID, err
	}

	return data.ID, nil
}
