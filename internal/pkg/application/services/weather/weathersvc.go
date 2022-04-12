package services

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"
)

type WeatherService interface {
	Start(ctx context.Context, log zerolog.Logger, authKey, lastChangeID, trafikverketURL, contextBrokerURL string) (string, error)
}

func NewWeatherService() WeatherService {
	return &ws{}
}

type ws struct {
}

func (ws *ws) Start(ctx context.Context, log zerolog.Logger, authKey, lastChangeID, trafikverketURL, contextBrokerURL string) (string, error) {

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
