package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	TopPriority  bool   `json:"topPriority"`
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

//Handler to recieve GET method sent on the /getItems endpoint
func getItems(c *gin.Context) {
	//Make initial connection with database
	sqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
	"password=%s dbname=%s sslmode=disable",
	host, port, user, password, dbname)
	db, err := sql.Open("postgres", sqlInfo)
	if err != nil {
		//Return an error if database is not active
		response := Response{Action: "SQL", Sucessful: false, Context: err.Error()}
		c.IndentedJSON(http.StatusOK, response)
	}
	db.Ping()

	//Grab all rows of the table associated with this program
	rows, sizeerr := db.Query("SELECT * FROM items")
	if (sizeerr != nil) {
		response := Response{Action: "SQL", Sucessful: false, Context: sizeerr.Error()}
		c.IndentedJSON(http.StatusOK, response)	
	}
	defer rows.Close()

	//Get any filters such as "Name" filters provided from Query Parameters and "ID" filters provided from path parameters
	nameFilter, nameFilterPresent := c.GetQuery("rawName")
	numFilter := c.Param("id")
	intNumFilter, _ := strconv.Atoi(c.Param("id"))
	//Check if ID parameter, if present, is greater than or equal to 0, else send an error
	if intNumFilter < 0 {
		response := Response{Action: "Get", Sucessful: false, Context: "ID cannot be below 0"}
		c.IndentedJSON(http.StatusOK, response)
	}
	//Declare variable for use in Row Scan
	var numFilterPresent bool = (numFilter != "")
	var collection Collection
	size := 0
	//Purpose of loop is to find how many items fit filters to know what size the array to send the data should be
	for rows.Next() {
		//Begin each loop by assuming it is a valid entry
		validEntry := true
		var entry Item
		//Read each row and translate that data to an Item struct
		rows.Scan(&entry.ID, &entry.Name, &entry.Desc, &entry.TopPriority, &entry.Completed)
		//name filter check
		if (nameFilterPresent) {
			if (entry.Name != nameFilter) {
				validEntry = false
			}
		}
		//ID filter check
		if (numFilterPresent) {
			if (entry.ID != intNumFilter) {
				validEntry = false
			}
		}
		//Runs when both checks are passed
		if (validEntry) {
			size++
		}
	}
	//Get new row data, as old row data will not iterate again
	newRows, sizeerr := db.Query("SELECT * FROM items")
	if (sizeerr != nil) {
		response := Response{Action: "SQL", Sucessful: false, Context: sizeerr.Error()}
		c.IndentedJSON(http.StatusOK, response)	
	}
	defer newRows.Close()

	//Assign variables to be used in Row scan
	index := 0
	collection.Items = make([]Item, size)
	for newRows.Next() {
		var entry Item
		var validEntry bool = true
		//Get the data and make it into a Item struct, it is same as the last loop
		newRows.Scan(&entry.ID, &entry.Name, &entry.Desc, &entry.TopPriority, &entry.Completed)
		//Name filter check
		if (nameFilterPresent) {
			if (entry.Name != nameFilter) {
				validEntry = false
			}
		}
		//ID filter check
		if (numFilterPresent) {
			columnID, _ := strconv.Atoi(numFilter)
			if (entry.ID != columnID) {
				validEntry = false
			}
		}
		//Runs if both checks are passed
		if (validEntry) {
			//Assigning passed entries into an array
			collection.Items[index] = entry
			index++
		}
	}

	//Sending out the array as json response data
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

/*
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
*/

//Handler for PATCH requests to /patchItem endpoint
func patchItem(c *gin.Context) {
	//Make database connection
	sqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
	"password=%s dbname=%s sslmode=disable",
	host, port, user, password, dbname)
	db, err := sql.Open("postgres", sqlInfo)
	if err != nil {
		//Send error on bad call
		response := Response{Action: "SQL", Sucessful: false, Context: err.Error()}
		c.IndentedJSON(http.StatusOK, response)
	}
	db.Ping()

	//Get data from database
	rows, pullErr := db.Query("SELECT * from items")
	if pullErr != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: pullErr.Error()}
		c.IndentedJSON(http.StatusOK, response)
	}
	defer rows.Close()

	//Search for correct ID
	found := false
	target, _ := strconv.Atoi(c.Param("id"))
	for rows.Next() {
		var entry Item
		//Make a new entry with data from the database
		rows.Scan(&entry.ID, &entry.Name, &entry.Desc, &entry.TopPriority, &entry.Completed)
		if (entry.ID == target) {
			found = true
			//Update Name value if present
			nameVal, nameFound := c.GetQuery("rawName")
			if (nameFound) {
				entry.Name = nameVal
			}
			//Update description value if present
			descVal, descFound := c.GetQuery("rawDesc")
			if (descFound) {
				entry.Desc = descVal
			}
			//Update priority value if present
			priorityVal, priorityFound := c.GetQuery("rawPriority")
			if (priorityFound) {
				entry.TopPriority = strings.EqualFold(priorityVal, "true")
			}
			//Update completed value if present
			completedVal, completedFound := c.GetQuery("rawCompleted") 
			if (completedFound) {
				entry.Completed = strings.EqualFold(completedVal, "true")
			}

			//Push to database
			db.Query(`
			UPDATE items
			SET name = $1, description = $2, priority = $3, completed = $4
			WHERE id = $5;`, entry.Name, entry.Desc, entry.TopPriority, entry.Completed, entry.ID)
		}
	}
	//Runs if could not find given ID
	if (!found) {
		response := Response{Action: "Patch", Sucessful: false, Context: "No entry found matching given ID"}
		c.IndentedJSON(http.StatusOK, response)
	} else {
		//Runs on success
		response := Response{Action: "Patch", Sucessful: true, Context: "Updated entry without error"}
		c.IndentedJSON(http.StatusOK, response)	
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
	router.PATCH("/updateItem/:id", patchItem)

	router.Run("0.0.0.0:8080")
}