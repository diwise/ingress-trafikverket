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
	"github.com/diwise/context-broker/pkg/ngsild/types"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
)

func (ws *weatherSvc) publishWeatherMeasurepointStatus(ctx context.Context, measurepoint weatherMeasurepoint) (err error) {
	_, span := tracer.Start(ctx, "publish-weatherobservations")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	var attributes []entities.EntityDecoratorFunc
	attributes, err = convertWeatherMeasurepointToFiwareEntity(measurepoint)

	if err != nil {
		err = fmt.Errorf("could not create attributes for weathermeasurepoint: %s", err.Error())
		return
	}

	fragment, _ := entities.NewFragment(attributes...)
	entityID := fiware.WeatherObservedIDPrefix + "se:trafikverket:api:weathermeasurepoint:" + measurepoint.ID

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	_, err = ws.ctxBrokerClient.MergeEntity(ctx, entityID, fragment, headers)
	if err != nil {
		if !errors.Is(err, ngsierrors.ErrNotFound) {
			err = fmt.Errorf("failed to merge entity: %s", err.Error())
			return
		}

		var entity types.Entity
		entity, err = entities.New(entityID, fiware.WeatherObservedTypeName, attributes...)
		if err != nil {
			err = fmt.Errorf("entities.New failed: %s", err.Error())
			return
		}

		_, err = ws.ctxBrokerClient.CreateEntity(ctx, entity, headers)
		if err != nil {
			err = fmt.Errorf("failed to post weather observed to context broker: %s", err.Error())
			return
		}
	}

	return nil
}

func convertWeatherMeasurepointToFiwareEntity(ws weatherMeasurepoint) ([]entities.EntityDecoratorFunc, error) {
	position := ws.Geometry.Position
	position = position[7 : len(position)-1]

	Longitude := strings.Split(position, " ")[0]
	newLong, _ := strconv.ParseFloat(Longitude, 32)
	Latitude := strings.Split(position, " ")[1]
	newLat, _ := strconv.ParseFloat(Latitude, 32)

	utcTime := ws.ModifiedTime.Format(time.RFC3339)

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 7),
		decorators.Location(newLat, newLong),
		decorators.Name(ws.Name),
		decorators.DateObserved(utcTime),
	)

	if ws.Observation.Air != nil {
		attributes = append(attributes,
			number("temperature", ws.Observation.Air.Temperature.Value, utcTime),
			number("humidity", ws.Observation.Air.RelativeHumidity.Value/100.0, utcTime),
		)
	}

	if len(ws.Observation.Wind) > 0 && ws.Observation.Wind[0].Direction != nil && ws.Observation.Wind[0].Speed != nil {
		attributes = append(
			attributes,
			number("windDirection", float64(ws.Observation.Wind[0].Direction.Value), utcTime),
			number("windSpeed", ws.Observation.Wind[0].Speed.Value, utcTime),
		)
	}

	return attributes, nil
}

func number(property string, value float64, at string) entities.EntityDecoratorFunc {
	return decorators.Number(property, value, properties.ObservedAt(at))
}
