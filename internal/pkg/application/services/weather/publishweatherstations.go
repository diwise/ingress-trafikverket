package weathersvc

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/diwise/context-broker/pkg/datamodels/fiware"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
)

func (ws *ws) publishWeatherStationStatus(ctx context.Context, weatherstation weatherStation) error {
	var err error

	_, span := tracer.Start(ctx, "publish-weatherobservations")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	attributes := convertWeatherStationToFiwareEntity(weatherstation)

	fragment, _ := entities.NewFragment(attributes...)
	entityID := fiware.WeatherObservedIDPrefix + "se:trafikverket:temp:" + weatherstation.ID

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	_, err = ws.ctxBrokerClient.MergeEntity(ctx, entityID, fragment, headers)
	if err != nil {
		if !errors.Is(err, ngsierrors.ErrNotFound) {
			ws.log.Error().Err(err).Msg("failed to merge entity")
		}
		entity, err := entities.New(entityID, fiware.WeatherObservedTypeName, attributes...)
		if err != nil {
			ws.log.Error().Err(err).Msg("entities.New failed")
		}

		_, err = ws.ctxBrokerClient.CreateEntity(ctx, entity, headers)
		if err != nil {
			ws.log.Error().Err(err).Msg("failed to post weather observed to context broker")
		}
	}

	return nil
}

func convertWeatherStationToFiwareEntity(ws weatherStation) []entities.EntityDecoratorFunc {
	position := ws.Geometry.Position
	position = position[7 : len(position)-1]

	Longitude := strings.Split(position, " ")[0]
	newLong, _ := strconv.ParseFloat(Longitude, 32)
	Latitude := strings.Split(position, " ")[1]
	newLat, _ := strconv.ParseFloat(Latitude, 32)

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 2),
		decorators.Location(newLat, newLong),
		decorators.Temperature(ws.Measurement.Air.Temp),
		decorators.DateObserved(ws.Measurement.MeasureTime),
	)

	return attributes
}
