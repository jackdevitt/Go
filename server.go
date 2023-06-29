package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"math/rand"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

const (
	host     = "172.17.0.3"
	port     = 5432
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

func connectDB() *sql.DB {
	//Establish database connection
	sqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, _ := sql.Open("postgres", sqlInfo)
	return db
}

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

func addItem(c *gin.Context) {
	db := connectDB()
	db.Ping()

	//Get info from parameters
	_, foundCompletion := c.GetQuery("rawCompleted")
	_, foundPriority := c.GetQuery("rawPriority")

	rand.Seed(time.Now().UnixNano())
	min := 1000000000
	max := 2147483647
	assignedID := rand.Intn(max - min + 1) + min
	//SQL command
	insertCMD := `INSERT INTO items (ID, Name, Description, Priority, Completed)
	VALUES ($1, $2, $3, $4, $5)`
	_, cmderr := db.Exec(insertCMD, assignedID, c.DefaultQuery("rawName", "Untitled"), c.DefaultQuery("rawDesc", "No Description"), foundPriority, foundCompletion)
	if cmderr != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: cmderr.Error()}
		c.IndentedJSON(http.StatusOK, response)
	} else {
		//Send message for successful POST method
		response := Response{Action: "Post", Sucessful: true, Context: "Posted without error"}
		c.IndentedJSON(http.StatusOK, response)
	}
}

//Handler to recieve GET method sent on the /getItems endpoint
func getItems(c *gin.Context) {
	db := connectDB()
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


//Handler for /removeItem endpoint with DELETE method
func removeItem(c *gin.Context) {
	db := connectDB()
	db.Ping()
	//Get value from Path
	identifier := c.Param("id")
	//SQL command
	_, err := db.Query(
		`DELETE FROM items
		WHERE id = $1;`, identifier)
	if err != nil {
		response := Response{Action: "SQL", Sucessful: false, Context: err.Error()}
		c.IndentedJSON(http.StatusOK, response)
	} else {
		//Send message for successful POST method
		response := Response{Action: "Delete", Sucessful: true, Context: "Removed without error"}
		c.IndentedJSON(http.StatusOK, response)
	}
}

//Handler for PATCH requests to /patchItem endpoint
func patchItem(c *gin.Context) {
	db := connectDB()
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

func main() {

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Type"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}

	router.Use(cors.New(config))

	router.GET("/health", checkHealth)
	router.POST("/addItem", addItem)
	router.DELETE("/removeItem/:id", removeItem)
	router.GET("/getItems/:id", getItems)
	router.GET("/getItems", getItems)
	router.PATCH("/updateItem/:id", patchItem)

	router.Run("0.0.0.0:8080")
}