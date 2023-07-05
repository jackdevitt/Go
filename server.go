package main

import (

	"fmt"
	"net/http"
	"strconv"
	
	"time"
	"strings"
	"encoding/json"
	"math/rand"
	"gorm.io/gorm"
	"gorm.io/driver/postgres"
	
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
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

type Item struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Description      string `json:"desc"`
	TopPriority  bool   `json:"topPriority"`
	Completed bool   `json:"completed"`
}

func connectDB() *gorm.DB {
	dsn := "host=localhost user=postgres password=password dbname=postgres port=8000 sslmode=disable TimeZone=America/New_York"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{});
	if err != nil {
		fmt.Print(err.Error());
	}
	db.AutoMigrate(&Item{});
	fmt.Print(db);
	return db
}
/*
func checkHealth(c *gin.Context) {
	sqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
	"password=%s dbname=%s sslmode=disable",
	host, port, user, password, dbname)
	db, _ := sql.Open("postgres", sqlInfo)
	db.Ping()
	_, err := db.Query(`
		SELECT * FROM items
		WHERE id = -1`)
	if (err == nil) {
		status := Response{Action: "Health", Sucessful: true, Context: "Systems are OK"}
		c.IndentedJSON(http.StatusOK, status)
	} else {
		status := Response{Action: "Health", Sucessful: false, Context: "Systems are DOWN"}
		c.IndentedJSON(http.StatusOK, status)
	}
}
*/
func addItem(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		response := Response{Action: "Post", Sucessful: false, Context: err.Error()}
		c.IndentedJSON(http.StatusBadRequest, response)
	}
	var entry Item
	json.Unmarshal(body, &entry)

	if (len(strings.TrimSpace(entry.Name)) == 0) {
		response := Response{Action: "Post", Sucessful: false, Context: "Please provide a Name"}
		c.IndentedJSON(http.StatusBadRequest, response);
	} else {

		rand.Seed(time.Now().UnixNano())
		entry.ID = rand.Intn(2147483647 - 1000000000) + 1000000000

		result := db.Create(&entry)
		if result.Error != nil {
			response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
			c.IndentedJSON(http.StatusBadRequest, response)
		} else {
			response := Response{Action: "Patch", Sucessful: true, Context: "Updated without error"}
			c.IndentedJSON(http.StatusOK, response)
		}
	}
}

//Handler to recieve GET method sent on the /getItems endpoint
func getItems(c *gin.Context) {
	var items []Item
	result := db.Find(&items)
	if result.Error != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
		c.IndentedJSON(http.StatusBadRequest, response)
	}
	nameFilter, nameFilterPresent := c.GetQuery("rawName")
	numFilter := c.Param("id")
	numFilterPresent := (numFilter != "")
	collection := Collection{Items: items}
	count := 0;
	idFilter,_ := strconv.Atoi(numFilter)
	if (numFilterPresent && (idFilter < 1000000000 || idFilter > 2147483647)) {
		response := Response{Action: "Post", Sucessful: false, Context: "ID given either too high or too low"}
		c.IndentedJSON(http.StatusBadRequest, response)
	} 
	for i := 0; i < len(collection.Items); i++ {
		validEntry := true;
		if (nameFilterPresent) {
			if (!strings.Contains(strings.ToLower(collection.Items[i].Name), strings.ToLower(nameFilter))) {
				validEntry = false;
			}
		}
		if (numFilterPresent) {
			if (collection.Items[i].ID != idFilter) {
				validEntry = false;
			}
		}
		if (validEntry) {
			count++;
		}
	}
	newCollection := make([]Item, count)
	index := 0
	for i := 0; i < len(collection.Items); i++ {
		validEntry := true;
		if (nameFilterPresent) {
			if (!strings.Contains(strings.ToLower(collection.Items[i].Name), strings.ToLower(nameFilter))) {
				validEntry = false;
			}
		}
		if (numFilterPresent) {
			idFilter, _ := strconv.Atoi(numFilter);
			if (collection.Items[i].ID != idFilter) {
				validEntry = false;
			}
		}
		if (validEntry) {
			newCollection[index] = collection.Items[i];
			index++;
		}
	}
	collection.Items = newCollection
	c.IndentedJSON(http.StatusOK, collection)
}


//Handler for /removeItem endpoint with DELETE method
func removeItem(c *gin.Context) {
	id := c.Param("id")
	if (id == "") {
		response := Response{Action: "Delete", Sucessful: false, Context: "Please enter an ID"}
		c.IndentedJSON(http.StatusBadRequest, response)
	}
	numId,_ := strconv.Atoi(id) 
	if (numId < 1000000000 || numId > 2147483647) {
		response := Response{Action: "Delete", Sucessful: false, Context: "ID given either too high or too low"}
		c.IndentedJSON(http.StatusBadRequest, response)
	}
	var entry Item
	result := db.First(&entry, numId)
	if result.Error != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: result.Error.Error()}
		c.IndentedJSON(http.StatusBadRequest, response)
	} else {
		db.Delete(&entry)
		response := Response{Action: "Delete", Sucessful: true, Context: "Deleted without error"}
		c.IndentedJSON(http.StatusOK, response)
	}
}


//Handler for PATCH requests to /patchItem endpoint
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
	} else {
		tableEntry.Name = entry.Name;
		tableEntry.Description = entry.Description;
		tableEntry.TopPriority = entry.TopPriority;
		tableEntry.Completed = entry.Completed;

		db.Save(&tableEntry)
	}

}

func main() {

	db = connectDB()
	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Type"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}

	router.Use(cors.New(config))

	//router.GET("/health", checkHealth)
	router.POST("/addItem", addItem)
	router.DELETE("/removeItem/:id", removeItem)
	router.GET("/getItems/:id", getItems)
	router.GET("/getItems", getItems)
	router.PATCH("/updateItem/:id", patchItem)

	router.Run("0.0.0.0:8080")
}