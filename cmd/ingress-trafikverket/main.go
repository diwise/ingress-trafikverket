package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/logging"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/tracing"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/geojson"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

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

func getAndPublishWeatherStationStatus(ctx context.Context, log zerolog.Logger, authKey, lastChangeID, trafikverketURL, contextBrokerURL string) (string, error) {

	responseBody, err := getWeatherStationStatus(ctx, trafikverketURL, authKey, lastChangeID)
	if err != nil {
		return lastChangeID, err
	}

	answer := &weatherStationResponse{}
	err = json.Unmarshal(responseBody, answer)
	if err != nil {
		return lastChangeID, err
	}

	for _, weatherstation := range answer.Response.Result[0].WeatherStations {
		err = publishWeatherStationStatus(ctx, weatherstation, contextBrokerURL)
		if err != nil {
			log.Error().Msgf("unable to publish data for weatherstation %s: %s", weatherstation.ID, err.Error())
		}
	}

	return answer.Response.Result[0].Info.LastChangeID, nil
}

func publishWeatherStationStatus(ctx context.Context, weatherstation weatherStation, contextBrokerURL string) error {

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
		return err
	}

	ctx, span := tracer.Start(ctx, "publish-weatherstation")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	url := fmt.Sprintf("%s/ngsi-ld/v1/entities/%s/attrs/", contextBrokerURL, device.ID)

	req, _ := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(patchBody))

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return err
	}

	return nil
}

var tracer = otel.Tracer("tfv-client")

func getWeatherStationStatus(ctx context.Context, trafikverketURL, authKey, lastChangeID string) ([]byte, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-weatherstations")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	log := logging.GetLoggerFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	requestBody := fmt.Sprintf("<REQUEST><LOGIN authenticationkey=\"%s\" /><QUERY objecttype=\"WeatherStation\" schemaversion=\"1\" changeid=\"%s\"><INCLUDE>Id</INCLUDE><INCLUDE>Geometry.WGS84</INCLUDE><INCLUDE>Measurement.Air.Temp</INCLUDE><INCLUDE>Measurement.MeasureTime</INCLUDE><INCLUDE>ModifiedTime</INCLUDE><INCLUDE>Name</INCLUDE><FILTER><WITHIN name=\"Geometry.SWEREF99TM\" shape=\"box\" value=\"527000 6879000, 652500 6950000\" /></FILTER></QUERY></REQUEST>", authKey, lastChangeID)

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodPost, trafikverketURL, bytes.NewBufferString(requestBody))
	if err != nil {
		log.Error().Err(err).Msg("failed to create http request")
		return []byte{}, err
	}
	apiReq.Header.Set("Content-Type", "text/xml")

	apiResponse, err := httpClient.Do(apiReq)
	if err != nil {
		log.Error().Msgf("failed to retrieve weatherstations")
		return nil, err
	}
	if apiResponse.StatusCode != http.StatusOK {
		log.Error().Msgf("expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
		return []byte{}, errors.New("")
	}

	defer apiResponse.Body.Close()

	responseBody, err := ioutil.ReadAll(apiResponse.Body)

	log.Info().Msgf("received response: " + string(responseBody))

	return responseBody, err
}

func main() {

	serviceVersion := version()
	serviceName := "ingress-trafikverket"

	ctx, logger := logging.NewLogger(context.Background(), serviceName, serviceVersion)
	logger.Info().Msg("starting up ...")

	cleanup, err := tracing.Init(ctx, logger, serviceName, serviceVersion)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init tracing")
	}
	defer cleanup()

	authenticationKey := getEnvironmentVariableOrDie(logger, "TFV_API_AUTH_KEY", "API Authentication Key")
	trafikverketURL := getEnvironmentVariableOrDie(logger, "TFV_API_URL", "API URL")
	contextBrokerURL := getEnvironmentVariableOrDie(logger, "CONTEXT_BROKER_URL", "Context Broker URL")

	lastChangeID := "0"

	for {
		lastChangeID, err = getAndPublishWeatherStationStatus(ctx, logger, authenticationKey, lastChangeID, trafikverketURL, contextBrokerURL)
		if err != nil {
			logger.Error().Msg(err.Error())
		}
		time.Sleep(30 * time.Second)
	}
}

func version() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	buildSettings := buildInfo.Settings
	infoMap := map[string]string{}
	for _, s := range buildSettings {
		infoMap[s.Key] = s.Value
	}

	sha := infoMap["vcs.revision"]
	if infoMap["vcs.modified"] == "true" {
		sha += "+"
	}

	return sha
}

func getEnvironmentVariableOrDie(log zerolog.Logger, envVar, description string) string {
	value := os.Getenv(envVar)
	if value == "" {
		log.Fatal().Msgf("Please set %s to a valid %s.", envVar, description)

	}
	return value
}
