package main

// TODO 1. Construct structs
// TODO 2. Check db for city
// TODO 3. If city is not in db or error was returned while searching db jump to step 5
// TODO 4. Return city from API
// TODO 5. Ask Chat if I need to run "defer db.Close()" on opening connection to db

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type CityStruct struct {
	ID        int     `json:"-"`
	Name      string  `json:"name"`
	Region    string  `json:"region"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	TZ_ID     string  `json:"tz_id"`
}

type ConditionStruct struct {
	Text string `json:"text"`
	Icon string `json:"icon"`
}

type CurrentStruct struct {
	ID            int             `json:"-"`
	CityID        int             `json:"-"`
	LastUpdated   string          `json:"last_updated"`
	TempC         float64         `json:"temp_c"`
	Condition     ConditionStruct `json:"condition"`
	WindMph       float64         `json:"wind_mph"`
	WindDegree    int             `json:"wind_degree"`
	WindDirection string          `json:"wind_dir"`
	PressureIn    float64         `json:"pressure_in"`
	Humidity      int             `json:"humidity"`
}

type ForecastStruct struct {
	ID          int                    `json:"-"`
	CityID      int                    `json:"-"`
	ForecastDay []ForecastDayArrStruct `json:"forecastday"`
}

type ForecastDayArrStruct struct {
	Date      string                  `json:"date"`
	DateEpoch int                     `json:"date_epoch"`
	Day       ForecastSingleDayStruct `json:"day"`
}

type ForecastSingleDayStruct struct {
	TempC      float32         `json:"avgtemp_c"`
	MaxWindMph float32         `json:"maxwind_mph"`
	Humidity   int             `json:"humidity"`
	Condition  ConditionStruct `json:"condition"`
}

// Database global variable
var Database *sql.DB

// Function to connect to database
func ConnectDB() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error occured while trying to load .env file")
	}
	db_host := os.Getenv("DB_HOST")
	db_port := os.Getenv("DB_PORT")
	db_user := os.Getenv("DB_USER")
	db_name := os.Getenv("DB_USER")
	db_password := os.Getenv("DB_PASSWORD")

	var uri string = fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable", db_host, db_port, db_user, db_name, db_password)

	db, err := sql.Open("postgres", uri)
	if err != nil {
		fmt.Println("Error occured while connecting to db")
	}
	Database = db
	fmt.Println("Database Connected Successfully")
}

// Databse connected successfully
