package main

import (

    "golang.org/x/crypto/bcrypt"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"os"
	"strings"
	"encoding/json"
	"math/rand"
	"gorm.io/gorm"
	"gorm.io/driver/postgres"
	
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"github.com/swaggo/gin-swagger" // gin-swagger middleware
    "github.com/swaggo/files" // swagger embed files
	_ "sample-app/docs"
)

var db *gorm.DB;

type Response struct {
	Action    string `json:"action"`
	Sucessful bool   `json:"sucessful"`
	Context   string `json:"context"`
}

type Collection struct {
	Items []Item `json:"Items"`
}

type Log struct {
	ID int `json:"id"`
	Severity string `json:"severity"`
	Content string `json:"content"`
	Timestamp string `json:"timestamp"`
}

type Item struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Description      string `json:"desc"`
	TopPriority  bool   `json:"topPriority"`
	Completed bool   `json:"completed"`
	UserID int `json:"userId"`
}

//lint:ignore U1000 Used for Swagger
type itemRequirements struct {
	Name      string `json:"name"`
	Description      string `json:"desc"`
	TopPriority  bool   `json:"topPriority"`
	Completed bool   `json:"completed"`
}

//lint:ignore U1000 Used for Swagger
type userRequirements struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type User struct {
	ID int `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func connectDB() *gorm.DB {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Failed to load .env")
	} else {
		dsn := os.Getenv("CONNSTRING")
		fmt.Println("Connection String: ", dsn)
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{});
		if err != nil {
			fmt.Println(err.Error());
		} else {
			db.AutoMigrate(&Item{});
			return db
		}
	}
	return nil
}

func log(severity string, content string) {
	var prevLog Log
	datetime := time.Now().Format(time.RFC3339)
	result := db.Last(&prevLog)

	if result.Error != nil {
		fmt.Println(result.Error.Error())
	}
	var log Log
	if prevLog.Severity != "" {
		log = Log{ID: prevLog.ID + 1, Severity: severity, Content: content, Timestamp: datetime}
	} else {
		log = Log{ID: 1, Severity: severity, Content: content, Timestamp: datetime}
	}
	createResult := db.Create(&log)

	if createResult.Error != nil {
		fmt.Println(createResult.Error.Error())
	}
}

func checkHealth(c *gin.Context) {
	var blank Item
	result := db.First(&blank)
	if (result.Error == nil) {
		status := Response{Action: "Health", Sucessful: true, Context: "Systems are OK"}
		c.IndentedJSON(http.StatusOK, status)
		log("INFO", "Systems up")
	} else {
		status := Response{Action: "Health", Sucessful: false, Context: "Systems are DOWN"}
		c.IndentedJSON(http.StatusOK, status)
		log("FATAL", "Systems down")
	}
}

// validateUser godoc
// @Summary validateUser
// @Description validate user with given username and passwod
// @Tags users
// @Param Item body userRequirements true "Item"
// @Accept */*
// @Produce json
// @Router /validateUser [POST]
func validateUser(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		response := Response{Action: "Validate", Sucessful: false, Context: err.Error()}
		c.IndentedJSON(http.StatusBadRequest, response)
		log("ERROR", "Validation failed")
	} else {
		var user User
		json.Unmarshal(body, &user)
		enteredPassword := user.Password
		fmt.Println(user)

		result := db.Where("username = ?", user.Username).First(&user)
		if result.Error != nil {
			response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
			c.IndentedJSON(http.StatusBadRequest, response) 
			log("ERROR", "Failed querying from SQL after Validation")
		} else {
			err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(enteredPassword))
			if err != nil {
				response := Response{Action: "Validate", Sucessful: false, Context: "Username or Password Incorrect"}
				c.IndentedJSON(http.StatusBadRequest, response)
				log("INFO", "Username and Password after Validation was incorrect")
			} else {
				c.IndentedJSON(http.StatusOK, user)	
				log("INFO", "Validation successful")
			}
		}
	}
}

// addUser godoc
// @Summary addUser
// @Description add user with given username and password
// @Tags users
// @Param Item body userRequirements true "Item"
// @Accept */*
// @Produce json
// @Router /addUser [POST]
func addUser(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		response := Response{Action: "Post", Sucessful: false, Context: err.Error()}
		c.IndentedJSON(http.StatusBadRequest, response)
		log("ERROR", "Adding user failed")
	}
	var user User
	json.Unmarshal(body, &user)
	if strings.TrimSpace(user.Password) == "" || strings.TrimSpace(user.Username) == "" {
		response := Response{Action: "Post", Sucessful: false, Context: "Please enter a username AND password"}
		c.IndentedJSON(http.StatusBadRequest, response)
		log("ERROR", "Missing information after adding user")
	} else {
		rand.Seed(time.Now().UnixNano())
		user.ID = rand.Intn(2147483647 - 1000000000) + 1000000000
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			response := Response{Action: "Encrypt", Sucessful: false, Context: err.Error()}
			c.IndentedJSON(http.StatusBadRequest, response)
			log("ERROR", "Encryption failed after adding user")
		} else {
			user.Password = string(hash)
			result := db.Create(&user)
			if result.Error != nil {
				response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
				c.IndentedJSON(http.StatusConflict, response)
				log("ERROR", "Failed querying from SQL after adding user")
			} else {
				c.IndentedJSON(http.StatusOK, user)
				log("INFO", "added user successfully")
			}
		}
	}
}

// addItem godoc
// @Summary addItem
// @Description add task to given user
// @Tags tasks
// @Param User-Id header string true "UserID" 
// @Param Item body itemRequirements true "Item"
// @Accept */*
// @Produce json
// @Router /addItem [POST]
func addItem(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		response := Response{Action: "Post", Sucessful: false, Context: err.Error()}
		c.IndentedJSON(http.StatusBadRequest, response)
		log("ERROR", "Query failed after adding item")
	}
	var entry Item
	json.Unmarshal(body, &entry)
	fmt.Println(entry)

	header := c.Request.Header["User-Id"]
	if len(header) == 0 {
		response := Response{Action: "Post", Sucessful: false, Context: "Please enter a target ID"}
		c.IndentedJSON(http.StatusBadRequest, response)
		log("ERROR", "Missing ID after adding item")
	} else {
		strUserId := header[0]
		intUserId,_ := strconv.Atoi(strUserId)
		entry.UserID = intUserId

		if (entry.UserID == 0) {
			response := Response{Action: "Post", Sucessful: false, Context: "Please enter a target ID"}
			c.IndentedJSON(http.StatusBadRequest, response)
			log("ERROR", "Missing ID after adding item")
		} else {
			if (len(strings.TrimSpace(entry.Name)) == 0) {
				response := Response{Action: "Post", Sucessful: false, Context: "Please provide a Name"}
				c.IndentedJSON(http.StatusBadRequest, response);
				log("ERROR", "Missing Name after adding item")
			} else {

				rand.Seed(time.Now().UnixNano())
				entry.ID = rand.Intn(2147483647 - 1000000000) + 1000000000

				result := db.Create(&entry)
				if result.Error != nil {
					response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
					c.IndentedJSON(http.StatusBadRequest, response)
					log("ERROR", "Failed querying from SQL after adding item")
				} else {
					response := Response{Action: "Patch", Sucessful: true, Context: "Updated without error"}
					c.IndentedJSON(http.StatusOK, response)
					log("INFO", "added item successfully")
				}
			}
		}
	}
}

// getItemById godoc
// @Summary getItemById
// @Description get tasks of given user ID and task ID
// @Tags tasks
// @Param User-Id header string true "UserID" 
// @Param id path int true "id"
// @Accept */*
// @Produce json
// @Router /getItemById/{id} [GET]
func getItemsById(c *gin.Context) {
	header := c.Request.Header["User-Id"]
	if len(header) == 0 {
		response := Response{Action: "Get", Sucessful: false, Context: "Please enter a target ID"}
		c.IndentedJSON(http.StatusBadRequest, response)
		log("ERROR", "Missing ID after getting Items (ID)")
	} else {
		body := header[0]
		if body == "" {
			response := Response{Action: "Get", Sucessful: false, Context: "Please enter a target ID"}
			c.IndentedJSON(http.StatusBadRequest, response)
			log("ERROR", "Missing ID after getting Items (ID)")
		} else {
			data := c.Param("id")
			if data == "" {
				response := Response{Action: "Get", Sucessful: false, Context: "Please enter a Item ID"}
				c.IndentedJSON(http.StatusBadRequest, response)
				log("ERROR", "Missing item ID after getting Items (ID)")
			} else {
				var collection Collection
				result := db.Where("id = ? AND user_Id = ?", data, body).Find(&collection.Items)
				if result.Error != nil {
					response := Response{Action: "Get", Sucessful: false, Context: result.Error.Error()}
					c.IndentedJSON(http.StatusBadRequest, response)
					log("ERROR", "Failed querying from SQL after getting Items (ID)")
				} else {
					c.IndentedJSON(http.StatusOK, collection)
					log("INFO", "Got Items (ID) successfully")
				}
			}
		}
	}
}

// getItemsByCount godoc
// @Summary getItemsByCount
// @Description get a certain number of tasks of given user ID
// @Tags tasks
// @Param User-Id header string true "UserID"  
// @Param rawName query string false "Filter" 
// @Param count query string true "Count" 
// @Accept */*
// @Produce json
// @Router /getItemsByCount [GET]
func getItemsByCount(c *gin.Context) {
	header := c.Request.Header["User-Id"]
	if len(header) == 0 {
		response := Response{Action: "Get", Sucessful: false, Context: "Please enter a target ID"}
		c.IndentedJSON(http.StatusBadRequest, response)
		log("ERROR", "Missing ID after getting Items")
	} else {
		body := header[0]
		if body == "" {
			response := Response{Action: "Get", Sucessful: false, Context: "Please enter a target ID"}
			c.IndentedJSON(http.StatusBadRequest, response)
			log("ERROR", "Missing ID after getting Items")
		} else {
			nameFilter, nameFilterPresent := c.GetQuery("rawName")
			nameFilter = "%" + nameFilter + "%"
			var collection Collection

			count,_ := strconv.Atoi(c.Query("count"))
			if (nameFilterPresent) {
				result := db.Limit(count).Where("name LIKE ? AND user_Id = ?", nameFilter, body).Find(&collection.Items)
				if result.Error != nil {
					response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
					c.IndentedJSON(http.StatusBadRequest, response)
					log("ERROR", "Failed querying from SQL after getting Items")
				} else {
					c.IndentedJSON(http.StatusOK, collection)
					log("INFO", "Got Items successfully")
				}
			} else {
				result := db.Limit(count).Where("user_Id = ?", body).Find(&collection.Items)
				if result.Error != nil {
					response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
					c.IndentedJSON(http.StatusBadRequest, response)
					log("ERROR", "Failed querying from SQL after getting Items")
				} else {
					c.IndentedJSON(http.StatusOK, collection)
					log("INFO", "Got Items successfully")
				}
			}
		}
	}
}

// getItems godoc
// @Summary getItems
// @Description get tasks of given user ID
// @Tags tasks
// @Param User-Id header string true "UserID"  
// @Param rawName query string false "Filter" 
// @Accept */*
// @Produce json
// @Router /getItems [GET]
func getItems(c *gin.Context) {

	header := c.Request.Header["User-Id"]
	if len(header) == 0 {
		response := Response{Action: "Get", Sucessful: false, Context: "Please enter a target ID"}
		c.IndentedJSON(http.StatusBadRequest, response)
		log("ERROR", "Missing ID after getting Items")
	} else {
		body := header[0]
		if body == "" {
			response := Response{Action: "Get", Sucessful: false, Context: "Please enter a target ID"}
			c.IndentedJSON(http.StatusBadRequest, response)
			log("ERROR", "Missing ID after getting Items")
		} else {
			nameFilter, nameFilterPresent := c.GetQuery("rawName")
			nameFilter = "%" + nameFilter + "%"
			var collection Collection
			if (nameFilterPresent) {
				result := db.Where("name LIKE ? AND user_Id = ?", nameFilter, body).Find(&collection.Items)
				if result.Error != nil {
					response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
					c.IndentedJSON(http.StatusBadRequest, response)
					log("ERROR", "Failed querying from SQL after getting Items")
				} else {
					c.IndentedJSON(http.StatusOK, collection)
					log("INFO", "Got Items successfully")
				}
			} else {
				result := db.Where("user_Id = ?", body).Find(&collection.Items)
				if result.Error != nil {
					response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
					c.IndentedJSON(http.StatusBadRequest, response)
					log("ERROR", "Failed querying from SQL after getting Items")
				} else {
					c.IndentedJSON(http.StatusOK, collection)
					log("INFO", "Got Items successfully")
				}
			}
		}
	}
}


// removeItem godoc
// @Summary removeItem
// @Description Remove tasks with given ID
// @Tags tasks
// @Param id path int true "id"
// @Accept */*
// @Produce json
// @Router /removeItem/{id} [DELETE]
func removeItem(c *gin.Context) {
	id := c.Param("id")
	if (id == "") {
		response := Response{Action: "Delete", Sucessful: false, Context: "Please enter an ID"}
		c.IndentedJSON(http.StatusBadRequest, response)
		log("ERROR", "Missing ID after removing Item")
	} else {
		numId,_ := strconv.Atoi(id) 
		if (numId < 1000000000 || numId > 2147483647) {
			response := Response{Action: "Delete", Sucessful: false, Context: "ID given either too high or too low"}
			c.IndentedJSON(http.StatusBadRequest, response)
			log("ERROR", "ID given cannot exist after removing Item")
		} else {
			var entry Item
			result := db.First(&entry, numId)
			if result.Error != nil {
				response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
				c.IndentedJSON(http.StatusBadRequest, response)
				log("ERROR", "Failed querying from SQL after removing Item")
			} else {
				db.Delete(&entry)
				response := Response{Action: "Delete", Sucessful: true, Context: "Deleted without error"}
				c.IndentedJSON(http.StatusOK, response)
				log("INFO", "Removed item successfully")
			}
		}
	}
}


// patchItem godoc
// @Summary patchItem
// @Description Update tasks with given ID
// @Tags tasks
// @Param id path int true "id"
// @Param Item body itemRequirements true "Name"
// @Accept */*
// @Produce json
// @Router /updateItem/{id} [PATCH]
func patchItem(c *gin.Context) {
	body,_ := c.GetRawData()
	var entry Item
	json.Unmarshal(body, &entry)

	id := c.Param("id")

	var tableEntry Item
	result := db.First(&tableEntry, id)
	if result.Error != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: "No entry matching given ID"}
		c.IndentedJSON(http.StatusBadRequest, response)
		log("ERROR", "Given ID does not exist after patching Item")
	} else {
		tableEntry.Name = entry.Name;
		tableEntry.Description = entry.Description;
		tableEntry.TopPriority = entry.TopPriority;
		tableEntry.Completed = entry.Completed;

		db.Save(&tableEntry)
		response := Response{Action: "Patch", Sucessful: false, Context: "Updated without error"}
		c.IndentedJSON(http.StatusOK, response)
		log("INFO", "Updated Item sucessfully")
	}
}

// @title Gin Swagger todo-list API
// @version 1.0
// @description This is a todo-list server.

// @host localhost:8080
// @BasePath /
// @schemes http
func main() {

	db = connectDB()
	if db != nil {
		log("INFO", "Connected to Database")
	}
	log("INFO", "Starting Program...")
	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Type", "User-Id"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}

	router.Use(cors.New(config))

	router.GET("/health", checkHealth)

	router.POST("/addItem", addItem)
	router.DELETE("/removeItem/:id", removeItem)
	router.GET("/getItemById/:id", getItemsById)
	router.GET("/getItemsByCount", getItemsByCount)
	router.GET("/getItems", getItems)
	router.PATCH("/updateItem/:id", patchItem)

	router.POST("/addUser", addUser)
	router.POST("/validateUser", validateUser)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.Run("0.0.0.0:8080")
}