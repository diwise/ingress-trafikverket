package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type WeatherService interface {
	Start(ctx context.Context) error
	getAndPublishWeatherStations(ctx context.Context, lastChangeID string) (string, error)
	getWeatherStationStatus(ctx context.Context, lastChangeID string) ([]byte, error)
	publishWeatherStationStatus(ctx context.Context, weatherstation weatherStation) error
}

func NewWeatherService(log zerolog.Logger, authKey, trafikverketURL, contextBrokerURL string) WeatherService {
	return &ws{
		log:               log,
		authenticationKey: authKey,
		trafikverketURL:   trafikverketURL,
		contextBrokerURL:  contextBrokerURL,
	}
}

type ws struct {
	log               zerolog.Logger
	authenticationKey string
	trafikverketURL   string
	contextBrokerURL  string
}

func (ws *ws) Start(ctx context.Context) error {
	var err error
	lastChangeID := "0"

	for {
		lastChangeID, err = ws.getAndPublishWeatherStations(ctx, lastChangeID)
		if err != nil {
			ws.log.Error().Msg(err.Error())
			return err
		}
		time.Sleep(30 * time.Second)
	}
}

func (ws *ws) getAndPublishWeatherStations(ctx context.Context, lastChangeID string) (string, error) {

	responseBody, err := ws.getWeatherStationStatus(ctx, lastChangeID)
	if err != nil {
		return lastChangeID, err
	}

	answer := &weatherStationResponse{}
	err = json.Unmarshal(responseBody, answer)
	if err != nil {
		return lastChangeID, err
	}

	for _, weatherstation := range answer.Response.Result[0].WeatherStations {
		err = ws.publishWeatherStationStatus(ctx, weatherstation)
		if err != nil {
			log.Error().Msgf("unable to publish data for weatherstation %s: %s", weatherstation.ID, err.Error())
		}
	}

	return answer.Response.Result[0].Info.LastChangeID, nil
}
