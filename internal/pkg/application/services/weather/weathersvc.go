package weathersvc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

type WeatherService interface {
	Start(ctx context.Context) error
	getAndPublishWeatherStations(ctx context.Context, lastChangeID string) (string, error)
	getWeatherStationStatus(ctx context.Context, lastChangeID string) ([]byte, error)
	publishWeatherStationStatus(ctx context.Context, weatherstation weatherStation) error
}

func NewWeatherService(log zerolog.Logger, authKey, trafikverketURL string, ctxBrokerClient client.ContextBrokerClient) WeatherService {
	return &ws{
		log:               log,
		authenticationKey: authKey,
		trafikverketURL:   trafikverketURL,
		ctxBrokerClient:   ctxBrokerClient,
		stations:          map[string]string{},
	}
}

type ws struct {
	log               zerolog.Logger
	authenticationKey string
	trafikverketURL   string
	ctxBrokerClient   client.ContextBrokerClient
	stations          map[string]string
}

func (ws *ws) Start(ctx context.Context) error {
	var err error
	lastChangeID := "0"

	for {
		lastChangeID, err = ws.getAndPublishWeatherStations(ctx, lastChangeID)
		if err != nil {
			ws.log.Error().Msg(err.Error())
		}
		time.Sleep(30 * time.Second)
	}
}

var tracer = otel.Tracer("tfv-weatherstation-client")

func (ws *ws) getAndPublishWeatherStations(ctx context.Context, lastChangeID string) (string, error) {
	var err error

	ctx, span := tracer.Start(ctx, "get-and-publish-status")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, ws.log, ctx)

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
		if weatherstation.Active {
			previousMeasureTime, ok := ws.stations[weatherstation.ID]
			if ok && previousMeasureTime == weatherstation.Measurement.MeasureTime {
				continue
			}

			ws.stations[weatherstation.ID] = weatherstation.Measurement.MeasureTime

			err = ws.publishWeatherStationStatus(ctx, weatherstation)
			if err != nil {
				log.Error().Err(err).Msgf("unable to publish data for weatherstation %s", weatherstation.ID)
			}
		}
	}

	return answer.Response.Result[0].Info.LastChangeID, nil
}
