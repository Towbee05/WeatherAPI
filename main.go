package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type LocationStruct struct {
	Id        uuid.UUID `json:"-"`
	Name      string    `json:"name"`
	Country   string    `json:"country"`
	Region    string    `json:"region"`
	Latitude  float64   `json:"lat"`
	Longitude float64   `json:"lon"`
	Tz_id     string    `json:"tz_id"`
}

type ConditionStruct struct {
	Text string `json:"text"`
	Icon string `json:"icon"`
}

type CurrentWeatherStruct struct {
	Id           uuid.UUID       `json:"-"`
	Condition    ConditionStruct `json:"condition"`
	Humidity     int             `json:"humidity"`
	Wind_mph     float64         `json:"wind_mph"`
	Wind_degree  int             `json:"wind_degree"`
	Pressure_in  float64         `json:"pressure_in"`
	Cloud        int             `json:"cloud"`
	Temp_celcius float64         `json:"temp_c"`
}

type WeatherResponseForCurrent struct {
	Location LocationStruct       `json:"location"`
	Current  CurrentWeatherStruct `json:"current"`
}

type ForecastMainContainer struct {
	Forecastday []ForecastDayContainerStruct `json:"forecastday"`
}

type ForecastDayContainerStruct struct {
	Date      string            `json:"date"`
	DateEpoch int               `json:"date_epoch"`
	Day       ForecastDayStruct `json:"day"`
}

type ForecastDayStruct struct {
	Id        int             `json:"-"`
	City_id   uuid.UUID       `json:"-"`
	Maxtemp_c float64         `json:"maxtemp_c"`
	Mintemp_c float64         `json:"mintemp_c"`
	Temp_c    float64         `json:"avgtemp_c"`
	Humidity  int             `json:"humidity"`
	Wind_mph  float64         `json:"maxwind_mph"`
	Condition ConditionStruct `json:"condition"`
}

type WeatherResponseForForecast struct {
	Location LocationStruct        `json:"location"`
	Current  CurrentWeatherStruct  `json:"current"`
	Forecast ForecastMainContainer `json:"forecast"`
}

var Database *sql.DB

func main() {
	env_err := godotenv.Load()
	if env_err != nil {
		// fmt.Println("An error occurred while loading env confuguration")
		fmt.Println("An error occurred while loading env confuguration")
	}
	var Api_key string = os.Getenv("WEATHER_API_KEY")
	fmt.Println("Hello, world")

	// Router
	var router *gin.Engine = gin.Default()
	connectDatabase()

	// GET request!!C
	router.GET("/current", func(context *gin.Context) {
		// Get query params, in this case "city" is the only query param we will be dealing with
		var city string = context.Query("city")
		var location LocationStruct
		var current CurrentWeatherStruct
		var weatherResponse WeatherResponseForCurrent
		// var condition ConditionStruct
		if city == "" {
			context.JSON(http.StatusBadGateway, gin.H{"message": "Please provide a city name"})
			return
		}
		// Check if city is in the database
		location_data__err := Database.QueryRow(`SELECT id, name, country, region, latitude, longitude, tz_id FROM city where name=$1`, city).Scan(&location.Id, &location.Name, &location.Country, &location.Region, &location.Longitude, &location.Latitude, &location.Tz_id)

		current_data__err := Database.QueryRow(`SELECT temp_celcius, humidity, wind_mph, condition_text, condition_icon, pressure_in, wind_dir, cloud FROM current where city_id=$1`, location.Id).Scan(&current.Temp_celcius, &current.Humidity, &current.Wind_mph, &current.Condition.Text, &current.Condition.Icon, &current.Pressure_in, &current.Wind_degree, &current.Cloud)
		if location_data__err != nil {
			fmt.Println("Location does not exist in Database.")
		}
		if current_data__err != nil {
			fmt.Println("Current does not exist in Database.")
		}

		if current_data__err == sql.ErrNoRows {
			fmt.Println("Current Table not added for city")
		}

		fmt.Println("Data from database ", location_data__err)
		fmt.Println("Location variable returns ", location)
		//If data is not in database
		if location_data__err == sql.ErrNoRows {
			var URL string = fmt.Sprintf("http://api.weatherapi.com/v1/current.json?key=%s&q=%s", Api_key, city)
			// Verify if external url was fetched successfully
			var weather, weather_err = http.Get(URL)
			if weather_err != nil {
				fmt.Println("An error occurred while tring to get URL")
			}
			// since "http.GET" cannot read the json content in the url, "io.ReadAll" parse the json data -I hope so
			var response, response_err = io.ReadAll(weather.Body)
			if response_err != nil {
				fmt.Println("An error occured while parsing weather body")
			}

			var json_response WeatherResponseForCurrent
			jsonErr := json.Unmarshal(response, &json_response)

			if jsonErr != nil {
				fmt.Println("An error occured while parsing weather body into JSON type")
			}
			id := uuid.New()

			// Create a location instance
			location = LocationStruct{
				Id:        id,
				Name:      json_response.Location.Name,
				Country:   json_response.Location.Country,
				Region:    json_response.Location.Region,
				Latitude:  json_response.Location.Latitude,
				Longitude: json_response.Location.Longitude,
				Tz_id:     json_response.Location.Tz_id,
			}
			condition := ConditionStruct{
				Text: json_response.Current.Condition.Text,
				Icon: json_response.Current.Condition.Icon,
			}
			// Create a current instance
			current = CurrentWeatherStruct{
				Id:           id,
				Temp_celcius: json_response.Current.Temp_celcius,
				Condition:    condition,
				// ConditionText: json_response.Current.ConditionText,
				// ConditionIcon: json_response.Current.ConditionIcon,
				Humidity:    json_response.Current.Humidity,
				Wind_mph:    json_response.Current.Wind_mph,
				Wind_degree: json_response.Current.Wind_degree,
				Pressure_in: json_response.Current.Pressure_in,
				Cloud:       json_response.Current.Cloud,
			}

			// Create a current weather instance
			weatherResponse = WeatherResponseForCurrent{
				Location: location,
				Current:  current,
			}
			// Add location to database
			InsertDataIntoCity(location, id)

			// Add current to database
			InsertDataIntoCurrent(id, current)

			context.JSON(200, weatherResponse)
			return
		} else {
			weatherResponse = WeatherResponseForCurrent{
				Location: location,
				Current:  current,
			}

			context.JSON(200, weatherResponse)
			return
		}
	})
	router.GET("/forecast", func(ctx *gin.Context) {
		var city string = ctx.Query("city")
		var city_id uuid.UUID
		var response WeatherResponseForForecast
		// var forecast ForecastDayStruct

		// Check DB to fetch city id
		city_id__err := Database.QueryRow(`SELECT id FROM city WHERE name=$1`, city).Scan(&city_id)

		if city_id__err != nil {
			fmt.Println("An error ocurred while fetching forecast from DB")
		}
		// Check if city exists in DB
		if city_id__err != sql.ErrNoRows {
			response = FetchForecastDataForCityFromApi(city, Api_key, "3")
			// InsertDataIntoCity()

		}

		ctx.JSON(200, response)
	})
	router.Run(":8080")
}

func connectDatabase() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("An error occurred while loading env confuguration")
	}
	// Load database configuration from .env file
	db_host := os.Getenv("DB_HOST")
	db_port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	db_user := os.Getenv("DB_USER")
	db_name := os.Getenv("DB_NAME")
	db_password := os.Getenv("DB_PASSWORD")

	// Setup database connection
	var URI string = fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable", db_host, db_port, db_user, db_name, db_password)

	db, sqlErr := sql.Open("postgres", URI)
	if sqlErr != nil {
		fmt.Println("Error connecting to database!!", err)
		panic(err)
	}
	Database = db
	fmt.Println("Database connected successfully")
}

func FetchForecastDataForCityFromApi(city string, ApiKey string, days string) WeatherResponseForForecast {
	var response WeatherResponseForForecast
	var url string = fmt.Sprintf("http://api.weatherapi.com/v1/forecast.json?key=%s&q=%s&days=%s", ApiKey, city, days)

	weatherHttpRequest, weatherHttpRequestErr := http.Get(url)
	if weatherHttpRequestErr != nil {
		fmt.Println("Error while requesting for data from URL")
		panic(weatherHttpRequestErr)
	}

	weatherBody, weatherBodyErr := io.ReadAll(weatherHttpRequest.Body)
	if weatherBodyErr != nil {
		fmt.Println("Error while parsing weather forecast body from URL")
		panic(weatherBodyErr)
	}

	jsonResponseAndErr := json.Unmarshal(weatherBody, &response)

	if jsonResponseAndErr != nil {
		fmt.Println("Unable to unmarshall forecast data into json")
	}
	return response
}

func InsertDataIntoCity(city LocationStruct, id uuid.UUID) {
	// Add city to DB
	_, err := Database.Exec(`INSERT INTO city (id, name, region, country, latitude, longitude, tz_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`, id, city.Name, city.Region, city.Country, city.Latitude, city.Longitude, city.Tz_id)

	if err != nil {
		fmt.Println("An error occured while inserting data into city")
		panic(err)
	}
}

func InsertDataIntoCurrent(cityId uuid.UUID, current CurrentWeatherStruct) {
	// Add current to DB
	_, err := Database.Exec(`INSERT INTO current (city_id, temp_celcius, humidity, wind_mph, condition_text, condition_icon, pressure_in, wind_dir, cloud) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, cityId, current.Temp_celcius, current.Humidity, current.Wind_mph, current.Condition.Text, current.Condition.Icon, current.Pressure_in, current.Wind_degree, current.Cloud)

	if err != nil {
		fmt.Println("An error occured while inserting data into city")
		panic(err)
	}
}

func InsertDataIntoForeCast(cityId uuid.UUID, forecast []ForecastDayContainerStruct) {
	// Add current to DB
	for _, day := range forecast {
		_, err := Database.Exec(`INSERT INTO forecast (maxtemp_c, mintemp_c, temp_c, humidity, wind_mph, date, condition_text, condition_icon, date_epoch) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, day.Day.Maxtemp_c, day.Day.Mintemp_c, day.Day.Temp_c, day.Day.Humidity, day.Day.Wind_mph, day.Date, day.Day.Condition.Text, day.Day.Condition.Icon, day.DateEpoch)

		if err != nil {
			fmt.Println("An error occured while inserting forecast data into DB")
			panic(err)
		}
	}

}
