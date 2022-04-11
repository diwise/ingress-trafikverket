package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/geojson"
	log "github.com/sirupsen/logrus"
)

// TFVError encapsulates a lower level error together with an error
// message provided by the caller that experienced the error
type TFVError struct {
	msg string
	err error
}

// FatalTFVError signals that an unrecoverable error has occured and that
// the calling application should terminate
type FatalTFVError struct {
	TFVError
}

// Error returns a concatenated error string
func (err *TFVError) Error() string {
	if err.err != nil {
		return err.msg + " (" + err.err.Error() + ")"
	}

	return err.msg
}

// NewError returns a new TFVError instance
func NewError(msg string, err error) *TFVError {
	return &TFVError{msg, err}
}

func (err *FatalTFVError) Error() string {
	return "FATAL: " + err.TFVError.Error()
}

// NewFatalError returns a new FatalTFVError instance
func NewFatalError(msg string, err error) *FatalTFVError {
	return &FatalTFVError{
		TFVError: TFVError{msg, err},
	}
}

type geometry struct {
	Position string `json:"WGS84"`
}

type measurement struct {
	Air         air    `json:"Air"`
	MeasureTime string `json:"MeasureTime"`
}

type air struct {
	Temp float64 `json:"Temp"`
}

type weatherStation struct {
	ID          string      `json:"ID"`
	Name        string      `json:"Name"`
	Geometry    geometry    `json:"Geometry"`
	Measurement measurement `json:"Measurement"`
}
type weatherStationResponse struct {
	Response struct {
		Result []struct {
			WeatherStations []weatherStation `json:"WeatherStation"`
			Info            struct {
				LastChangeID string `json:"LASTCHANGEID"`
			} `json:"INFO"`
		} `json:"RESULT"`
	} `json:"RESPONSE"`
}

func getAndPublishWeatherStationStatus(authKey, lastChangeID, trafikverketURL, contextBrokerURL string) (string, error) {

	responseBody, err := getWeatherStationStatus(trafikverketURL, authKey, lastChangeID)
	if err != nil {
		return lastChangeID, NewError("failed to retrieve weather station status: %s", err)
	}

	answer := &weatherStationResponse{}
	err = json.Unmarshal(responseBody, answer)
	if err != nil {
		return lastChangeID, NewError("unable to unmarshal response", err)
	}

	for _, weatherstation := range answer.Response.Result[0].WeatherStations {
		err = publishWeatherStationStatus(weatherstation, contextBrokerURL)
		if err != nil {
			log.Errorf("unable to publish data for weatherstation %s: %s", weatherstation.ID, err.Error())
		}
		log.Infof("successfully sent patch for %s to context broker", weatherstation.ID)
	}

	return answer.Response.Result[0].Info.LastChangeID, nil
}

func publishWeatherStationStatus(weatherstation weatherStation, contextBrokerURL string) error {

	position := weatherstation.Geometry.Position
	position = position[7 : len(position)-1]

	Longitude := strings.Split(position, " ")[0]
	newLong, _ := strconv.ParseFloat(Longitude, 32)
	Latitude := strings.Split(position, " ")[1]
	newLat, _ := strconv.ParseFloat(Latitude, 32)

	device := fiware.NewDevice("se:trafikverket:temp:"+weatherstation.ID, fmt.Sprintf("t=%.1f", weatherstation.Measurement.Air.Temp))
	device.Location = geojson.CreateGeoJSONPropertyFromWGS84(newLong, newLat)

	patchBody, err := json.Marshal(device)
	if err != nil {
		return NewError("failed to marshal telemetry message", err)
	}

	url := fmt.Sprintf("%s/ngsi-ld/v1/entities/%s/attrs/", contextBrokerURL, device.ID)

	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(patchBody))

	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return NewError("request to context broker failed", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return NewError(fmt.Sprintf("context broker returned status code %d", resp.StatusCode), nil)
	}

	return nil
}

func getWeatherStationStatus(trafikverketURL, authKey, lastChangeID string) ([]byte, error) {
	requestBody := fmt.Sprintf("<REQUEST><LOGIN authenticationkey=\"%s\" /><QUERY objecttype=\"WeatherStation\" schemaversion=\"1\" changeid=\"%s\"><INCLUDE>Id</INCLUDE><INCLUDE>Geometry.WGS84</INCLUDE><INCLUDE>Measurement.Air.Temp</INCLUDE><INCLUDE>Measurement.MeasureTime</INCLUDE><INCLUDE>ModifiedTime</INCLUDE><INCLUDE>Name</INCLUDE><FILTER><WITHIN name=\"Geometry.SWEREF99TM\" shape=\"box\" value=\"527000 6879000, 652500 6950000\" /></FILTER></QUERY></REQUEST>", authKey, lastChangeID)

	apiResponse, err := http.Post(
		trafikverketURL,
		"text/xml",
		bytes.NewBufferString(requestBody),
	)

	if err != nil {
		return []byte{}, NewError("failed to request weather station data from Trafikverket", err)
	}

	if apiResponse.StatusCode != http.StatusOK {
		return []byte{}, NewError(fmt.Sprintf("trafikverket returned status code %d", apiResponse.StatusCode), nil)
	}

	defer apiResponse.Body.Close()

	responseBody, err := ioutil.ReadAll(apiResponse.Body)

	log.Info("received response: " + string(responseBody))

	return responseBody, err
}

func main() {

	serviceName := "ingress-trafikverket"

	log.SetFormatter(&log.JSONFormatter{})
	log.Infof("Starting up %s ...", serviceName)

	authenticationKey := getEnvironmentVariableOrDie("TFV_API_AUTH_KEY", "API Authentication Key")
	trafikverketURL := getEnvironmentVariableOrDie("TFV_API_URL", "API URL")
	contextBrokerURL := getEnvironmentVariableOrDie("CONTEXT_BROKER_URL", "Context Broker URL")

	lastChangeID := "0"
	var err error = nil

	for {
		lastChangeID, err = getAndPublishWeatherStationStatus(authenticationKey, lastChangeID, trafikverketURL, contextBrokerURL)
		if err != nil {
			switch err.(type) {
			case *FatalTFVError:
				log.Fatal(err)
			default:
				log.Error(err)
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func getEnvironmentVariableOrDie(envVar, description string) string {
	value := os.Getenv(envVar)
	if value == "" {
		log.Fatalf("Please set %s to a valid %s.", envVar, description)
	}
	return value
}
