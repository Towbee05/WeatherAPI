package dto

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Towbee05/weather-service/internal/db"
	// main "github.com/Towbee05/weather-service/cmd/api"
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

var Database *sql.DB = db.ConnectDB()

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

// Function to save forecast data to database
func SaveForecastDataToDatabase(data WeatherResponseForForecast) (int, error) {
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

func FetchForecastDataFromAPI(cityName string) (WeatherResponseForForecast, bool, error) {
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
