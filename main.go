package main

// TODO 1. Construct structs
// TODO 2. Check db for city
// TODO 3. If city is not in db or error was returned while searching db jump to step 5
// TODO 4. Return city from API
// TODO 5. Ask Chat if I need to run "defer db.Close()" on opening connection to db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
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
	Humidity   int             `json:"avghumidity"`
	Condition  ConditionStruct `json:"condition"`
}

type WeatherResponseForCurrent struct {
	Location CityStruct    `json:"location"`
	Current  CurrentStruct `json:"current"`
}

type WeatherResponseForForecast struct {
	Location CityStruct     `json:"location"`
	Current  CurrentStruct  `json:"current"`
	Forecast ForecastStruct `json:"forecast"`
}

// Database global variable
var Database *sql.DB
var ctx = context.Background()

func main() {
	ConnectDB()
	// Connect to redis cache for in-memory storage
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		Protocol: 2,
	})
	var router *gin.Engine = gin.Default()
	err := godotenv.Load()
	if err != nil {
		fmt.Println("could not access .env files")
		return
	}
	// Engine is running now, check if the redis server is reciving requests
	pong, err := redisClient.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Could not connect to redis server")
	} else {
		fmt.Printf("Connected to redis server: %s", pong)
	}
	// Current URL
	router.GET("/current", func(context *gin.Context) {
		city := context.Query("city")
		if city == "" || city == " " {
			context.JSON(400, gin.H{
				"status":  400,
				"message": "Please provide a city name",
				"error":   "Bad Request",
			})
			return
		}

		// Check item in cache first
		itemData, hit := getItemFromCache(*redisClient, city)
		var data WeatherResponseForCurrent
		if hit {
			json.Unmarshal([]byte(itemData), &data)
			context.JSON(200, data)
			return
		}

		_, found, err := CheckLocationInDB(city)
		if err != nil {
			context.JSON(500, gin.H{
				"status":  500,
				"message": err,
				"error":   "Internal server error",
			})
		}
		if found == false && err == nil {
			apiResponse, _, _ := fetchForecastDataFromAPI(city)
			_, err := saveForecastDataToDatabase(apiResponse)
			if err != nil {
				context.JSON(500, gin.H{
					"status":  500,
					"message": err,
					"error":   "Internal Server Error",
				})
				return
			}
			data := WeatherResponseForCurrent{
				Location: apiResponse.Location,
				Current:  apiResponse.Current,
			}
			setItemToCache(*redisClient, city, &data, nil)
			context.JSON(200, data)
			return
		}
		response := FetchCurrentDataFromDB(city)
		setItemToCache(*redisClient, city, &response, nil)
		context.JSON(200, response)
	})
	// Forecast URL
	router.GET("/forecast", func(context *gin.Context) {
		// context.JSON(200, gin.H{"message": "Hello, world"})
		city := context.Query("city")
		if city == "" || city == " " {
			context.JSON(400, gin.H{
				"status":  400,
				"message": "Please provide a city name",
				"error":   "Bad Request",
			})
			return
		}
		// Check item in Cache
		_, found, err := CheckLocationInDB(city)
		if err != nil {
			context.JSON(404, gin.H{
				"status":  500,
				"message": "Internal server error",
				"error":   "Internal server error",
			})
		}
		if found == false && err == nil {
			apiResponse, _, _ := fetchForecastDataFromAPI(city)
			_, err := saveForecastDataToDatabase(apiResponse)
			if err != nil {
				context.JSON(500, gin.H{
					"status":  500,
					"message": err,
					"error":   "Internal Server Error",
				})
				return
			}
			context.JSON(200, apiResponse)
			return
		}
		response := FetchForecastDataFromDB(city)
		context.JSON(200, response)
	})

	router.Run(":8080")
}

// Function to connect to database
func ConnectDB() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error occured while trying to load .env file")
	}
	db_host := os.Getenv("DB_HOST")
	db_port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	db_user := os.Getenv("DB_USER")
	db_name := os.Getenv("DB_NAME")
	db_password := os.Getenv("DB_PASSWORD")

	var uri string = fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable", db_host, db_port, db_user, db_name, db_password)

	db, err := sql.Open("postgres", uri)
	if err != nil {
		fmt.Println("Error occured while connecting to db")
		panic(err)
	}
	Database = db
	fmt.Println("Database Connected Successfully")
	// Databse connected successfully
}

// Checking if location exists in the database; if not fetch from api
func CheckLocationInDB(city string) (int, bool, error) {
	var cityId int
	err := Database.QueryRow(`SELECT id FROM location WHERE name=$1`, city).Scan(&cityId)

	if err == sql.ErrNoRows {
		fmt.Println("Could not find city in DB")
		return 0, false, nil
	}
	if err != nil {
		fmt.Println("An error occured while trying to access city id from DB")
		return 0, false, err
	}
	return cityId, true, nil

}

func FetchCurrentDataFromDB(cityID string) WeatherResponseForCurrent {
	var location CityStruct
	var current CurrentStruct
	locationErr := Database.QueryRow(`SELECT id, name, region, country, latitude, longitude, tz_id FROM location WHERE name=$1`, cityID).Scan(
		&location.ID, &location.Name, &location.Region, &location.Country, &location.Latitude, &location.Longitude, &location.TZ_ID,
	)
	currentErr := Database.QueryRow(`SELECT id, city_id, last_updated, temp_c, condition_text, condition_icon, wind_mph, wind_degree, pressure_in, humidity FROM current WHERE city_id=$1`, location.ID).Scan(
		&current.ID, &current.CityID, &current.LastUpdated, &current.TempC, &current.Condition.Text, &current.Condition.Icon, &current.WindMph, &current.WindDegree, &current.PressureIn, &current.Humidity,
	)
	if locationErr == sql.ErrNoRows {
		fmt.Println("City does not exist in DB: FetchForecastDataFromDB")
	}
	if locationErr != nil {
		fmt.Println("An error occured while fetching from DB: FetchForecastDataFromDB")
	}
	if currentErr == sql.ErrNoRows {
		fmt.Println("City does not exist in DB: FetchForecastDataFromDB")
	}
	if currentErr != nil {
		fmt.Println("An error occured while fetching from DB: FetchForecastDataFromDB")
	}
	var response = WeatherResponseForCurrent{
		Location: location,
		Current:  current,
	}
	return response
}

func FetchForecastDataFromDB(cityID string) WeatherResponseForForecast {
	var location CityStruct
	var current CurrentStruct
	var forecast ForecastStruct

	locationErr := Database.QueryRow(`SELECT id, name, region, country, latitude, longitude, tz_id FROM location WHERE name=$1`, cityID).Scan(
		&location.ID, &location.Name, &location.Region, &location.Country, &location.Latitude, &location.Longitude, &location.TZ_ID,
	)
	currentErr := Database.QueryRow(`SELECT id, city_id, last_updated, temp_c, condition_text, condition_icon, wind_mph, wind_degree, pressure_in, humidity FROM current WHERE city_id=$1`, location.ID).Scan(
		&current.ID, &current.CityID, &current.LastUpdated, &current.TempC, &current.Condition.Text, &current.Condition.Icon, &current.WindMph, &current.WindDegree, &current.PressureIn, &current.Humidity,
	)
	rows, rowsErr := Database.Query(`SELECT date, date_epoch, temperature, wind_mph, humidity, condition_text, condition_icon FROM forecast WHERE city_id=$1`, location.ID)
	for rows.Next() {
		var f ForecastDayArrStruct
		rows.Scan(
			&f.Date, &f.DateEpoch, &f.Day.TempC, &f.Day.MaxWindMph, &f.Day.Humidity, &f.Day.Condition.Text, &f.Day.Condition.Icon,
		)
		forecast.ForecastDay = append(forecast.ForecastDay, f)
	}
	if rowsErr != nil {
		fmt.Println("City does not exist in DB: FetchForecastDataFromDB")
	}
	if locationErr == sql.ErrNoRows {
		fmt.Println("City does not exist in DB: FetchForecastDataFromDB")
	}
	if locationErr != nil {
		fmt.Println("An error occured while fetching from DB: FetchForecastDataFromDB")
	}
	if currentErr == sql.ErrNoRows {
		fmt.Println("City does not exist in DB: FetchForecastDataFromDB")
	}
	if currentErr != nil {
		fmt.Println("An error occured while fetching from DB: FetchForecastDataFromDB")
	}
	var response = WeatherResponseForForecast{
		Location: location,
		Current:  current,
		Forecast: forecast,
	}
	return response
}

func fetchForecastDataFromAPI(cityName string) (WeatherResponseForForecast, bool, error) {
	var apiKey string = os.Getenv("WEATHER_API_KEY")
	var url string = fmt.Sprintf("http://api.weatherapi.com/v1/forecast.json?key=%s&q=%s&days=3", apiKey, cityName)
	var response WeatherResponseForForecast
	var location CityStruct
	var current CurrentStruct
	var forecast ForecastStruct
	endpoint, err := http.Get(url)
	if err != nil {
		fmt.Println("An error ocurred while fetching data from API")
	}
	// fmt.Println(endpoint)
	readData, readErr := io.ReadAll(endpoint.Body)
	if readErr != nil {
		fmt.Println(readErr)
		response = WeatherResponseForForecast{
			Location: location,
			Current:  current,
			Forecast: forecast,
		}
		return response, false, readErr
	}
	json.Unmarshal(readData, &response)
	return response, true, nil
}

// Function to save forecast data to database
func saveForecastDataToDatabase(data WeatherResponseForForecast) (int, error) {
	location := data.Location
	current := data.Current
	forecastDay := data.Forecast.ForecastDay

	_, err := Database.Exec("INSERT INTO location (name, region, country, latitude, longitude, tz_id) VALUES ($1, $2, $3, $4, $5, $6)", location.Name, location.Region, location.Country, location.Latitude, location.Longitude, location.TZ_ID)
	if err != nil {
		fmt.Println("An error occured while adding location to db")
		return 0, err
	}

	cityId, _, _ := CheckLocationInDB(location.Name)
	_, currErr := Database.Exec("INSERT INTO current (city_id, last_updated, temp_c, condition_text, condition_icon, wind_mph, wind_degree, pressure_in, humidity, wind_dir) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)", cityId, current.LastUpdated, current.TempC, current.Condition.Text, current.Condition.Icon, current.WindMph, current.WindDegree, current.PressureIn, current.Humidity, current.WindDirection)
	if currErr != nil {
		fmt.Println("An error occured while adding current to db")
		return 0, currErr
	}
	for _, value := range forecastDay {
		_, forecastErr := Database.Exec("INSERT INTO forecast ( city_id, date, date_epoch, temperature, wind_mph, humidity, condition_text, condition_icon) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", cityId, value.Date, value.DateEpoch, value.Day.TempC, value.Day.MaxWindMph, value.Day.Humidity, value.Day.Condition.Text, value.Day.Condition.Icon)
		if forecastErr != nil {
			fmt.Println("An error occured while forecast location to db")
			return 0, forecastErr
		}
	}
	return 1, nil
}

// Function to retrieve item from the cache
func getItemFromCache(client redis.Client, key string) (string, bool) {
	value, err := client.Get(ctx, key).Result()
	if err != nil {
		return "An error occured while getting data from cache", false
	} else {
		return value, true
	}
}

func setItemToCache(client redis.Client, key string, current *WeatherResponseForCurrent, forecast *WeatherResponseForForecast) string {
	var marshalledData any
	if current != nil {
		marshalledData, _ = json.Marshal(current)
	}
	if forecast != nil {
		marshalledData, _ = json.Marshal(forecast)
	}
	err := client.Set(ctx, key, marshalledData, 0)
	if err != nil {
		return "An error occured while setting data to cache"
	} else {
		return "Item set to cache"
	}
}
