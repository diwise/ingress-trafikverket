package roadaccidents

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
)

func (ts *ts) publishRoadAccidentToContextBroker(ctx context.Context, dev tfvDeviation, deleted bool) error {
	var err error
	ctx, span := tracer.Start(ctx, "publish-to-broker")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	logger := logging.GetFromContext(ctx)

	attributes, err := convertRoadAccidentToFiwareEntity(dev, deleted)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create attribute for fiware entity")
		return err
	}

	fragment, _ := entities.NewFragment(attributes...)
	entityID := fiware.RoadAccidentIDPrefix + dev.Id

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	_, err = ts.ctxBroker.MergeEntity(ctx, entityID, fragment, headers)
	if err != nil {
		if !errors.Is(err, ngsierrors.ErrNotFound) {
			logger.Error().Err(err).Msg("failed to merge entity")
			return err
		}

		entity, err := entities.New(entityID, fiware.RoadAccidentTypeName, attributes...)
		if err != nil {
			logger.Error().Err(err).Msg("entities.New failed")
			return err
		}

		_, err = ts.ctxBroker.CreateEntity(ctx, entity, headers)
		if err != nil {
			logger.Error().Err(err).Msg("failed to post road accident to context broker")
			return err
		}
	}

	return nil
}

func convertRoadAccidentToFiwareEntity(ra tfvDeviation, deleted bool) ([]entities.EntityDecoratorFunc, error) {
	status := map[bool]string{
		true:  "solved",
		false: "onGoing",
	}

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 6),
		decorators.Description(ra.Message),
		decorators.Status(status[deleted]),
	)

	if ra.Geometry.WGS84 != "" {
		lat, lon := getLocationFromString(ra.Geometry.WGS84)
		attributes = append(attributes, decorators.Location(lat, lon))
	}

	if ra.StartTime != "" {
		t, _ := time.Parse(time.RFC3339, ra.StartTime)
		utcTime := t.UTC().Format(time.RFC3339)
		attributes = append(attributes, decorators.DateCreated(utcTime), decorators.DateTime("accidentDate", utcTime))
	}

	return attributes, nil
}

func getLocationFromString(location string) (latitude float64, longitude float64) {
	position := location[7 : len(location)-1]

	Longitude := strings.Split(position, " ")[0]
	newLong, _ := strconv.ParseFloat(Longitude, 32)
	Latitude := strings.Split(position, " ")[1]
	newLat, _ := strconv.ParseFloat(Latitude, 32)

	return newLat, newLong
}
