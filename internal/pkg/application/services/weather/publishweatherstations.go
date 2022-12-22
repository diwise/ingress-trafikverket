package weathersvc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/context-broker/pkg/datamodels/fiware"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
)

func (ws *ws) publishWeatherStationStatus(ctx context.Context, weatherstation weatherStation) error {
	var err error

	_, span := tracer.Start(ctx, "publish-weatherobservations")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	attributes, err := convertWeatherStationToFiwareEntity(weatherstation)
	if err != nil {
		ws.log.Error().Err(err).Msgf("could not create attributes for weatherstation")
	}

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

func convertWeatherStationToFiwareEntity(ws weatherStation) ([]entities.EntityDecoratorFunc, error) {
	position := ws.Geometry.Position
	position = position[7 : len(position)-1]

	Longitude := strings.Split(position, " ")[0]
	newLong, _ := strconv.ParseFloat(Longitude, 32)
	Latitude := strings.Split(position, " ")[1]
	newLat, _ := strconv.ParseFloat(Latitude, 32)

	convertedTime, err := convertTimeToRFC3339Format(ws.Measurement.MeasureTime)
	if err != nil {
		return nil, err
	}

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 7),
		decorators.Location(newLat, newLong),
		decorators.Name(ws.Name),
		number("temperature", ws.Measurement.Air.Temp, convertedTime),
		number("humidity", ws.Measurement.Air.RelativeHumidity/100.0, convertedTime),
		decorators.DateObserved(convertedTime),
	)

	if ws.Measurement.Wind.Direction != 0 || ws.Measurement.Wind.Force > 0.01 {
		attributes = append(
			attributes,
			number("windDirection", float64(ws.Measurement.Wind.Direction), convertedTime),
			number("windSpeed", ws.Measurement.Wind.Force, convertedTime),
		)
	}

	return attributes, nil
}

func convertTimeToRFC3339Format(timestring string) (string, error) {
	layout := "2006-01-02T15:04:05.999-07:00"
	parsedTime, err := time.Parse(layout, timestring)
	if err != nil {
		return "", fmt.Errorf("failed to parse time from string: %s", err.Error())
	}

	formattedTime := parsedTime.UTC().Format(time.RFC3339)

	return formattedTime, nil
}

func number(property string, value float64, at string) entities.EntityDecoratorFunc {
	return decorators.Number(property, value, properties.ObservedAt(at))
}
