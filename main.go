package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/spf13/viper"
)

func invalidZipCodeResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnprocessableEntity)
	w.Write([]byte("invalid zipcode"))
}

func zipCodeNotFoundResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("cannot find zipcode"))
}

func internalServerErrorResponse(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

func GetLocationByZipCode(zipCode string) (string, error) {
	request, err := http.NewRequest("GET", "https://viacep.com.br/ws/"+zipCode+"/json/", nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", nil
	}

	jsonResponse := make(map[string]string)

	err = json.NewDecoder(response.Body).Decode(&jsonResponse)
	if err != nil {
		return "", err
	}

	return jsonResponse["localidade"], nil
}

func GetTemperatureByLocation(location string, apiKey string) (float64, error) {
	uri := "https://api.weatherapi.com/v1/current.json"
	uri += "?q=" + url.QueryEscape(location)

	request, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}

	request.Header.Set("accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("key", apiKey)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		rawResponse, _ := io.ReadAll(response.Body)
		fmt.Println(string(rawResponse))

		return 0, nil
	}

	jsonResponse := make(map[string]interface{})
	err = json.NewDecoder(response.Body).Decode(&jsonResponse)
	if err != nil {
		return 0, err
	}

	current, ok := jsonResponse["current"].(map[string]interface{})
	if !ok {
		return 0, nil
	}

	tempC, ok := current["temp_c"].(float64)
	if !ok {
		return 0, nil
	}

	return tempC, nil
}

type conf struct {
	WeatherApiKey string `mapstructure:"WEATHER_API_KEY"`
}

func LoadConfig(path string) (*conf, error) {
	var cfg *conf
	viper.SetConfigName("app_config")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		panic(err)
	}
	return cfg, err
}

func handler(w http.ResponseWriter, r *http.Request) {
	cfg, err := LoadConfig(".")
	if err != nil {
		internalServerErrorResponse(w, err)
		return
	}

	cep := r.URL.Query().Get("cep")

	if cep == "" || len(cep) != 8 {
		invalidZipCodeResponse(w)
		return
	}

	location, err := GetLocationByZipCode(cep)
	if err != nil {
		internalServerErrorResponse(w, err)
		return
	}

	if location == "" {
		zipCodeNotFoundResponse(w)
		return
	}

	temperatureC, err := GetTemperatureByLocation(location, cfg.WeatherApiKey)
	if err != nil {
		fmt.Println(err)
		internalServerErrorResponse(w, err)
		return
	}

	tempK := temperatureC + 273.15
	tempF := (temperatureC * 9 / 5) + 32

	response := map[string]interface{}{
		"temp_C": temperatureC,
		"temp_K": tempK,
		"temp_F": tempF,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/temp", handler)

	fmt.Println("Server running on port 8080")
	http.ListenAndServe(":8080", nil)
}
