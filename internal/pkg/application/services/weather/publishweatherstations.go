package weathersvc

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/context-broker/pkg/datamodels/fiware"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
)

func (ws *ws) publishWeatherStationStatus(ctx context.Context, weatherstation weatherStation) error {
	var err error

	_, span := tracer.Start(ctx, "publish-weatherobservations")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	logger := logging.GetFromContext(ctx)

	attributes, err := convertWeatherStationToFiwareEntity(weatherstation)
	if err != nil {
		logger.Error("could not create attributes for weatherstation", "err", err.Error())
	}

	fragment, _ := entities.NewFragment(attributes...)
	entityID := fiware.WeatherObservedIDPrefix + "se:trafikverket:temp:" + weatherstation.ID

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	_, err = ws.ctxBrokerClient.MergeEntity(ctx, entityID, fragment, headers)
	if err != nil {
		if !errors.Is(err, ngsierrors.ErrNotFound) {
			logger.Error("failed to merge entity", "err", err.Error())
			return err
		}
		entity, err := entities.New(entityID, fiware.WeatherObservedTypeName, attributes...)
		if err != nil {
			logger.Error("entities.New failed", "err", err.Error())
			return err
		}

		_, err = ws.ctxBrokerClient.CreateEntity(ctx, entity, headers)
		if err != nil {
			logger.Error("failed to post weather observed to context broker", "err", err.Error())
			return err
		}
	}

	return nil
}

func convertWeatherStationToFiwareEntity(ws weatherStation) ([]entities.EntityDecoratorFunc, error) {
	position := ws.Geometry.Position
	position = position[7 : len(position)-1]

	Longitude := strings.Split(position, " ")[0]
	newLong, _ := strconv.ParseFloat(Longitude, 32)
	Latitude := strings.Split(position, " ")[1]
	newLat, _ := strconv.ParseFloat(Latitude, 32)

	t, _ := time.Parse(time.RFC3339, ws.Measurement.MeasureTime)
	utcTime := t.UTC().Format(time.RFC3339)

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 7),
		decorators.Location(newLat, newLong),
		decorators.Name(ws.Name),
		number("temperature", ws.Measurement.Air.Temp, utcTime),
		number("humidity", ws.Measurement.Air.RelativeHumidity/100.0, utcTime),
		decorators.DateObserved(utcTime),
	)

	if ws.Measurement.Wind.Direction != 0 || ws.Measurement.Wind.Force > 0.01 {
		attributes = append(
			attributes,
			number("windDirection", float64(ws.Measurement.Wind.Direction), utcTime),
			number("windSpeed", ws.Measurement.Wind.Force, utcTime),
		)
	}

	return attributes, nil
}

func number(property string, value float64, at string) entities.EntityDecoratorFunc {
	return decorators.Number(property, value, properties.ObservedAt(at))
}
