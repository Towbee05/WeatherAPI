package db

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Function to connect to database
func ConnectDB() *sql.DB {
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
	Database := db
	fmt.Println("Database Connected Successfully")
	// Databse connected successfully
	return Database
}
