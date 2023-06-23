package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"encoding/json"
	"os"
	"fmt"
	"strconv"
)

type Response struct {
	Action string `json:"Action"`
	Sucessful bool `json:"Sucessful"`
	Context string `json:"Context"`
}

type healthStatus struct {
	Status string `json:"status"`
}

type Collection struct {
	Items []Item `json:"Items"`
}

type Item struct {
	ID int `json:"ID"`
	Name string `json:"Name"`
	Desc string `json:"Desc"` 
	Completed bool `json:"Completed"`
}

func checkHealth(c *gin.Context) {
	var status = healthStatus{"OK"}
	c.IndentedJSON(http.StatusOK, status)
}

func addItem(c *gin.Context) {
	_, found := c.GetQuery("rawCompleted")
	var collection Collection
	var str = readFile("list.json")
	json.Unmarshal([]byte(str), &collection)
	var entry = Item{ID: len(collection.Items) + 1, Name: c.DefaultQuery("rawName", "Untitled"), Desc: c.DefaultQuery("rawDesc", "No Description"), Completed: found}
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
	c.IndentedJSON(http.StatusOK, collection)
}

func removeItem(c *gin.Context) {
	index,_ := strconv.Atoi(c.Query("rawID"))
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
	fmt.Print(newArr)
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
		fmt.Print("FATAL ERROR: CANNOT WRITE")
	} else {
		file.WriteString(data)
	}
}

func main() {

	router := gin.Default()
	router.GET("/health", checkHealth)

	router.POST("/addItem", addItem)
	router.DELETE("/removeItem", removeItem)
	router.GET("/getItems", getItems)

	router.Run("localhost:8080")
}