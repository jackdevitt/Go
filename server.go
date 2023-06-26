package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 8000
	user     = "postgres"
	password = "password"
	dbname   = "postgres"
)

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
	Desc      string `json:"desc"`
	TopPriority  bool   `json:"priority"`
	Completed bool   `json:"completed"`
}

func checkHealth(c *gin.Context) {
	var status = Response{Action: "Health", Sucessful: true, Context: "Systems ARE GOOD"}
	c.IndentedJSON(http.StatusOK, status)
}

func addItem(c *gin.Context) {
	sqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", sqlInfo)
	if err != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: err.Error()}
		c.IndentedJSON(http.StatusOK, response)
	}
	defer db.Close()
	db.Ping()

	_, foundCompletion := c.GetQuery("rawCompleted")
	_, foundPriority := c.GetQuery("rawPriority")

	length, sizeerr := db.Query("SELECT * FROM items")
	counter := 1
	for length.Next() {
		counter++
	}
	if sizeerr != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: sizeerr.Error()}
		c.IndentedJSON(http.StatusOK, response)
	}
	insertCMD := `INSERT INTO items (ID, Name, Description, Priority, Completed)
	VALUES ($1, $2, $3, $4, $5)`
	_, cmderr := db.Exec(insertCMD, counter, c.DefaultQuery("rawName", "Untitled"), c.DefaultQuery("rawDesc", "No Description"), foundPriority, foundCompletion)
	if cmderr != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: cmderr.Error()}
		c.IndentedJSON(http.StatusOK, response)
	}
}

func getItems(c *gin.Context) {
	sqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
	"password=%s dbname=%s sslmode=disable",
	host, port, user, password, dbname)
	db, err := sql.Open("postgres", sqlInfo)
	if err != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: err.Error()}
		c.IndentedJSON(http.StatusOK, response)
	}
	db.Ping()

	rows, sizeerr := db.Query("SELECT * FROM items")
	if (sizeerr != nil) {
		response := Response{Action: "SQL", Sucessful: false, Context: sizeerr.Error()}
		c.IndentedJSON(http.StatusOK, response)	
	}
	defer rows.Close()

	nameFilter, nameFilterPresent := c.GetQuery("rawName")
	numFilter := c.Param("id")
	intNumFilter, _ := strconv.Atoi(c.Param("id"))
	if intNumFilter < 0 {
		response := Response{Action: "Get", Sucessful: false, Context: "ID cannot be below 0"}
		c.IndentedJSON(http.StatusOK, response)
	}
	var numFilterPresent bool = (numFilter != "")
	var collection Collection
	size := 0
	for rows.Next() {
		validEntry := true
		var entry Item
		rows.Scan(&entry.ID, &entry.Name, &entry.Desc, &entry.TopPriority, &entry.Completed)
		if (nameFilterPresent) {
			if (entry.Name != nameFilter) {
				validEntry = false
			}
		}
		if (numFilterPresent) {
			if (entry.ID != intNumFilter) {
				validEntry = false
			}
		}
		if (validEntry) {
			size++
		}
	}
	newRows, sizeerr := db.Query("SELECT * FROM items")
	if (sizeerr != nil) {
		response := Response{Action: "SQL", Sucessful: false, Context: sizeerr.Error()}
		c.IndentedJSON(http.StatusOK, response)	
	}
	defer newRows.Close()

	index := 0
	collection.Items = make([]Item, size)
	for newRows.Next() {
		var entry Item
		var validEntry bool = true
		newRows.Scan(&entry.ID, &entry.Name, &entry.Desc, &entry.TopPriority, &entry.Completed)
		if (nameFilterPresent) {
			if (entry.Name != nameFilter) {
				validEntry = false
			}
		}
		if (numFilterPresent) {
			columnID, _ := strconv.Atoi(numFilter)
			if (entry.ID != columnID) {
				validEntry = false
			}
		}
		if (validEntry) {
			collection.Items[index] = entry
			index++
		}
	}

	c.IndentedJSON(http.StatusOK, collection)
}

/*func getItems(c *gin.Context) {
	var collection Collection
	var str = readFile("list.json")
	json.Unmarshal([]byte(str), &collection)
	filter, status := c.GetQuery("rawFilter")
	if !status {
		c.IndentedJSON(http.StatusOK, collection)
	} else {
		count := 0
		for i := 0; i < len(collection.Items); i++ {
			if strings.EqualFold(collection.Items[i].Name, filter) {
				count++
			}
		}
		if count != 0 {
			index := 0
			newArr := make([]Item, count)
			for i := 0; i < len(collection.Items); i++ {
				if strings.EqualFold(collection.Items[i].Name, filter) {
					newArr[index] = collection.Items[i]
					index++
				}
			}
			collection.Items = newArr
			c.IndentedJSON(http.StatusOK, collection.Items)
		} else {
			response := Response{Action: "Get", Sucessful: false, Context: "No queries match filter"}
			c.IndentedJSON(http.StatusOK, response)
		}
	}
}*/

func removeItem(c *gin.Context) {
	_, status := c.GetQuery("rawID")
	index, _ := strconv.Atoi(c.Query("rawID"))
	if !status {
		response := Response{Action: "Delete", Sucessful: false, Context: "No ID entered"}
		c.IndentedJSON(http.StatusOK, response)
	} else {
		var str = readFile("list.json")
		var collection Collection
		json.Unmarshal([]byte(str), &collection)
		if index > len(collection.Items) || index <= 0 {
			response := Response{Action: "Delete", Sucessful: false, Context: "Index out of bounds"}
			c.IndentedJSON(http.StatusOK, response)
		} else {
			for i := 0; i < len(collection.Items); i++ {
				if collection.Items[i].ID == index {
					collection.Items = removeArrayItem(collection.Items, i)
					break
				}
			}
			newStr, _ := json.MarshalIndent(collection, "", "    ")
			writeToFile("list.json", string(newStr))
			response := Response{Action: "Delete", Sucessful: true, Context: "Deleted without error"}
			c.IndentedJSON(http.StatusOK, response)
		}
	}
}

func replaceItem(c *gin.Context) {
	_, status := c.GetQuery("rawID")
	index, _ := strconv.Atoi(c.Query("rawID"))
	if !status {
		response := Response{Action: "Patch", Sucessful: false, Context: "No ID entered"}
		c.IndentedJSON(http.StatusOK, response)
	} else {
		var collection Collection
		var str = readFile("list.json")
		json.Unmarshal([]byte(str), &collection)
		if index > len(collection.Items) || index < 0 {
			response := Response{Action: "Patch", Sucessful: false, Context: "Index out of bounds"}
			c.IndentedJSON(http.StatusOK, response)
		} else {
			_, foundCompletion := c.GetQuery("rawCompleted")
			_, foundPriority := c.GetQuery("rawPriority")
			entry := Item{ID: len(collection.Items) + 1, Name: c.DefaultQuery("rawName", "Untitled"), Desc: c.DefaultQuery("rawDesc", "No Description"), Completed: foundCompletion, TopPriority: foundPriority}
			collection.Items = replaceArrayItem(collection.Items, index, entry)
			newStr, _ := json.MarshalIndent(collection, "", "    ")
			writeToFile("list.json", string(newStr))
			response := Response{Action: "Patch", Sucessful: true, Context: "Updated without error"}
			c.IndentedJSON(http.StatusOK, response)
		}
	}
}

func sqlf(c *gin.Context) {
	sqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", sqlInfo)
	if err != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: err.Error()}
		c.IndentedJSON(http.StatusOK, response)
	}
	db.Ping()
}

func removeArrayItem(arr []Item, index int) []Item {
	newArr := make([]Item, len(arr)-1)
	newIndex := 0
	for i := 0; i < len(arr); i++ {
		if i != index {
			newArr[newIndex] = arr[i]
			newArr[newIndex].ID = newIndex + 1
			newIndex++
		}
	}
	return newArr
}

func replaceArrayItem(arr []Item, index int, replacement Item) []Item {
	newArr := make([]Item, len(arr))
	for i := 0; i < len(arr); i++ {
		if i+1 == index {
			newArr[i] = replacement
			newArr[i].ID = i + 1
		} else {
			newArr[i] = arr[i]
		}
	}
	return newArr
}

func readFile(fileName string) string {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Sprintf("FATAL ERROR: %d", err)
	} else {
		return string(file)
	}
}

func writeToFile(fileName, data string) {
	os.Truncate(fileName, 0)
	file, err := os.OpenFile(fileName, os.O_WRONLY, 0600)
	if err != nil {
		fmt.Print("FATAL ERROR: ", err)
	} else {
		file.WriteString(data)
	}
}

func main() {

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Type"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}

	router.Use(cors.New(config))

	router.GET("/health", checkHealth)
	router.GET("/SQL", sqlf)
	router.POST("/addItem", addItem)
	router.DELETE("/removeItem", removeItem)
	router.GET("/getItems/:id", getItems)
	router.GET("/getItems", getItems)
	router.PATCH("/replaceItem", replaceItem)

	router.GET("/item", getItems)
	router.POST("/item", addItem)
	router.DELETE("/item", removeItem)
	router.PATCH("/item", replaceItem)

	router.Run("0.0.0.0:8080")
}
