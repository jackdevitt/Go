package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"encoding/json"
	"os"
	"fmt"
	"strconv"
	"strings"
	"github.com/gin-contrib/cors"
)

type Response struct {
	Action string `json:"Action"`
	Sucessful bool `json:"Sucessful"`
	Context string `json:"Context"`
}

type Collection struct {
	Items []Item `json:"Items"`
}

type Item struct {
	ID int `json:"ID"`
	Name string `json:"Name"`
	Desc string `json:"Desc"` 
	Priority bool `json:"Priority"`
	Completed bool `json:"Completed"`
}

func checkHealth(c *gin.Context) {
	var status = Response{Action: "Health", Sucessful: true, Context: "Systems OK"}
	c.IndentedJSON(http.StatusOK, status)
}

func addItem(c *gin.Context) {
	_, foundCompletion := c.GetQuery("rawCompleted")
	_, foundPriority := c.GetQuery("rawPriority")
	var collection Collection
	var str = readFile("list.json")
	json.Unmarshal([]byte(str), &collection)
	var entry = Item{ID: len(collection.Items) + 1, Name: c.DefaultQuery("rawName", "Untitled"), Desc: c.DefaultQuery("rawDesc", "No Description"), Completed: foundCompletion, Priority: foundPriority}
	collection.Items = append(collection.Items, entry)
	newStr,_ := json.MarshalIndent(collection, "", "    ")
	writeToFile("list.json", string(newStr))
	response := Response{Action: "Post", Sucessful: true, Context: "Posted without error"}
	c.IndentedJSON(http.StatusOK, response)
}

func getItems(c *gin.Context) {
	var collection Collection
	var str = readFile("list.json")
	json.Unmarshal([]byte(str), &collection)
	filter, status := c.GetQuery("rawFilter")
	if (!status) {
		c.IndentedJSON(http.StatusOK, collection)
	} else {
		count := 0
		for i := 0; i < len(collection.Items); i++ {
			if (strings.EqualFold(collection.Items[i].Name, filter)) {
				count++;
			}
		}
		if (count != 0) {
			index := 0
			newArr := make([]Item, count)
			for i := 0; i < len(collection.Items); i++ {
				if (strings.EqualFold(collection.Items[i].Name, filter)) {
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
}

func removeItem(c *gin.Context) {
	_, status := c.GetQuery("rawID")
	index, _ := strconv.Atoi(c.Query("rawID"))
	if (!status) {
		response := Response{Action: "Delete", Sucessful: false, Context: "No ID entered"}
		c.IndentedJSON(http.StatusOK, response)
	} else {
		var str = readFile("list.json")
		var collection Collection
		json.Unmarshal([]byte(str), &collection)
		if (index > len(collection.Items) || index <= 0) {
			response := Response{Action: "Delete", Sucessful: false, Context: "Index out of bounds"}
			c.IndentedJSON(http.StatusOK, response) 
		} else {
			for i := 0; i < len(collection.Items); i++ {
				if (collection.Items[i].ID == index) {
					collection.Items = removeArrayItem(collection.Items, i)
					break
				}
			}
			newStr,_ := json.MarshalIndent(collection, "", "    ")
			writeToFile("list.json", string(newStr))
			response := Response{Action: "Delete", Sucessful: true, Context: "Deleted without error"}
			c.IndentedJSON(http.StatusOK, response)
		}
	}
}

func replaceItem(c *gin.Context) {
	_, status := c.GetQuery("rawID")
	index, _ := strconv.Atoi(c.Query("rawID"))
	if (!status) {
			response := Response{Action: "Patch", Sucessful: false, Context: "No ID entered"}
			c.IndentedJSON(http.StatusOK, response)
	} else {
		var collection Collection
		var str = readFile("list.json")
		json.Unmarshal([]byte(str), &collection)
		if (index > len(collection.Items) || index < 0) {
			response := Response{Action: "Patch", Sucessful: false, Context: "Index out of bounds"}
			c.IndentedJSON(http.StatusOK, response)
		} else {
			_, foundCompletion := c.GetQuery("rawCompleted")
			_, foundPriority := c.GetQuery("rawPriority")
			entry := Item{ID: len(collection.Items) + 1, Name: c.DefaultQuery("rawName", "Untitled"), Desc: c.DefaultQuery("rawDesc", "No Description"), Completed: foundCompletion, Priority: foundPriority}
			collection.Items = replaceArrayItem(collection.Items, index, entry)
			newStr,_ := json.MarshalIndent(collection, "", "    ")
			writeToFile("list.json", string(newStr))
			response := Response{Action: "Patch", Sucessful: true, Context: "Updated without error"}
			c.IndentedJSON(http.StatusOK, response)
		}
	}
}

func removeArrayItem(arr []Item, index int) []Item {
	newArr := make([]Item, len(arr) - 1)
	newIndex := 0
	for i := 0; i < len(arr); i++ {
		if (i != index) {
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
		if (i + 1 == index) {
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
	if (err != nil) {
		return fmt.Sprintf("FATAL ERROR: %d", err)
	} else {
		return string(file)
	}
}

func writeToFile(fileName, data string) {
	os.Truncate(fileName, 0)
	file, err := os.OpenFile(fileName, os.O_WRONLY, 0600)
	if (err != nil) {
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

	router.POST("/addItem", addItem)
	router.DELETE("/removeItem", removeItem)
	router.GET("/getItems", getItems)
	router.PATCH("/replaceItem", replaceItem)

	router.GET("/item", getItems)
	router.POST("/item", addItem)
	router.DELETE("/item", removeItem)
	router.PATCH("/item", replaceItem)

	router.Run("localhost:8080")
}