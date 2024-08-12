package common

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"slices"
	"strings"
)

// Env is a globally-accessible variable that holds the environment variables
// for the application. It is initialized in the init function. Just import it in your
// package and use it to access the environment variables.
var Env = EnvVars{}

func init() {
	initEnv()
}

type EnvVars struct {
	ENVIRONMENT string `json:"ENVIRONMENT" type:"enum" options:"development,production,test"`
	BASE_URL    string `json:"BASE_URL" default:"http://localhost:3000"`
	PORT        string `json:"PORT" default:"3000"`

	// * Add more environment variables here
}

type Environment struct {
	Environment string  `json:"environment" type:"enum" options:"development,production,test"`
	Production  EnvVars `json:"production"`
	Development EnvVars `json:"development"`
	Test        EnvVars `json:"test"`
}

func initEnv() {
	// get env.json file
	envFile, err := os.Open("env.json")
	if err != nil {
		log.Fatalf("Error opening env.json: %v", err)
	}
	defer envFile.Close()

	// read env.json file
	envBytes, err := io.ReadAll(envFile)
	if err != nil {
		log.Fatalf("Error reading env.json: %v", err)
	}

	// unmarshal env.json file into Environment struct
	var envConfig Environment
	err = json.Unmarshal(envBytes, &envConfig)
	if err != nil {
		log.Fatalf("Error unmarshalling env.json: %v", err)
	}

	// extract the correct environment variables
	environmentType := envConfig.Environment
	var envVars EnvVars
	if environmentType == "production" {
		envVars = envConfig.Production
	} else if environmentType == "development" {
		envVars = envConfig.Development
	} else if environmentType == "test" {
		envVars = envConfig.Test
	} else {
		log.Fatalf("Invalid environment type: %s", environmentType)
	}

	// validate the environment variables based on the struct tags
	log.Println("===============VALIDATING ENVIRONMENT VARIABLES===============")
	envVars, err = validateEnvVars(envVars)
	if err != nil {
		log.Fatalf("Error validating environment variables: %v", err)
	}
	log.Println("===============VALIDATED ENVIRONMENT VARIABLES===============")

	// set the global Env variable
	Env = envVars
}

func validateEnvVars(envVars EnvVars) (EnvVars, error) {
	val := reflect.ValueOf(&envVars).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		varName, ok := field.Tag.Lookup("json")
		if !ok {
			continue
		}
		varVal := val.Field(i).String()
		if varVal == "" {
			varVal, ok = field.Tag.Lookup("default")
			if !ok {
				return EnvVars{}, fmt.Errorf("environment variable %s not found", varName)
			}
		}

		if field.Tag.Get("type") == "enum" {
			optionsString := field.Tag.Get("options")
			options := strings.Split(optionsString, ",")
			if !slices.Contains(options, varVal) {
				return EnvVars{}, fmt.Errorf("invalid value for %s. Expected one of: '%s'; got '%s'", varName, optionsString, varVal)
			}
		}
		log.Println(varName, varVal)
		val.Field(i).SetString(varVal)
	}
	return envVars, nil
}
