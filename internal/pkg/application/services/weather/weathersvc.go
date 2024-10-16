package weathersvc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel"
)

type WeatherService interface {
	Start(ctx context.Context) error
	getAndPublishWeatherStations(ctx context.Context, lastChangeID string) (string, error)
	getWeatherStationStatus(ctx context.Context, lastChangeID string) ([]byte, error)
	publishWeatherStationStatus(ctx context.Context, weatherstation weatherStation) error
}

func NewWeatherService(ctx context.Context, authKey, trafikverketURL string, ctxBrokerClient client.ContextBrokerClient) WeatherService {
	return &ws{
		authenticationKey: authKey,
		trafikverketURL:   trafikverketURL,
		ctxBrokerClient:   ctxBrokerClient,
		stations:          map[string]string{},
	}
}

type ws struct {
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
			logging.GetFromContext(ctx).Error(
				"failed to get and publish weather stations", "err", err.Error(),
			)
		}
		time.Sleep(30 * time.Second)
	}
}

var tracer = otel.Tracer("tfv-weatherstation-client")

func (ws *ws) getAndPublishWeatherStations(ctx context.Context, lastChangeID string) (string, error) {
	var err error

	ctx, span := tracer.Start(ctx, "get-and-publish-status")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(
		span, logging.GetFromContext(ctx), ctx,
	)

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
				log.Error("unable to publish data for weatherstation", "weatherstation", weatherstation.ID, "err", err)
			}
		}
	}

	return answer.Response.Result[0].Info.LastChangeID, nil
}
