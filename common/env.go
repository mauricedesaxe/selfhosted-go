package common

import (
	"log"
	"os"
	"reflect"

	"github.com/joho/godotenv"
)

// Env is a globally-accessible variable that holds the environment variables
// for the application. It is initialized in the init function. Just import it in your
// package and use it to access the environment variables.
var Env = Environment{}

func init() {
	Env.init()
}

type Environment struct {
	// Use the `env` tag to specify the name of the environment variable.
	// Use the `default` tag to specify a default value. If a variable is not found
	// and a default value is not specified, the application will panic.

	// Application settings
	ENVIRONMENT string `env:"ENVIRONMENT" default:"production"` // development, production, test
	BASE_URL    string `env:"BASE_URL" default:"http://localhost:3000"`
	PORT        string `env:"PORT" default:"3000"`

	// * Add more environment variables here
}

func (e *Environment) init() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	val := reflect.ValueOf(e).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		envVar, ok := field.Tag.Lookup("env")
		if !ok {
			continue
		}
		envValue := os.Getenv(envVar)
		if envValue == "" {
			envValue, ok = field.Tag.Lookup("default")
			if !ok {
				log.Panicf("Environment variable %s not found", envVar)
			}
			log.Printf("Using default value for %s: %s", envVar, envValue)
		}
		val.Field(i).SetString(envValue)
	}
}
