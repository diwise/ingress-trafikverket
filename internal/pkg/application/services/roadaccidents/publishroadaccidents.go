package roadaccidents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/geojson"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var ErrAlreadyExists = errors.New("already exists")

func (ts *ts) publishRoadAccidentToContextBroker(ctx context.Context, dev tfvDeviation) error {
	var err error
	ctx, span := tracer.Start(ctx, "publish-to-broker")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	logger := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	ra := fiware.NewRoadAccident(dev.Id)
	if dev.StartTime != "" {
		t, _ := time.Parse(time.RFC3339, dev.StartTime)
		utcTime := t.UTC().Format(time.RFC3339)

		ra.AccidentDate = *ngsitypes.CreateDateTimeProperty(utcTime)
		ra.DateCreated = ra.AccidentDate
	}

	if dev.Geometry.WGS84 != "" {
		ra.Location = getLocationFromString(dev.Geometry.WGS84)
	}

	ra.Description = *ngsitypes.NewTextProperty(dev.Message)
	ra.Status = *ngsitypes.NewTextProperty("onGoing")

	requestBody, err := json.Marshal(ra)
	if err != nil {
		return err
	}

	cbUrl := fmt.Sprintf("%s/ngsi-ld/v1/entities", ts.contextBrokerURL)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cbUrl, bytes.NewBuffer(requestBody))
	req.Header.Add("Content-Type", "application/ld+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		if resp.StatusCode != http.StatusConflict {
			errMsg := fmt.Sprintf("failed to send road accident to context broker, expected status code %d, but got %d", http.StatusOK, resp.StatusCode)
			return errors.New(errMsg)
		}

		cbUrl = fmt.Sprintf("%s/ngsi-ld/v1/entities/%s/attrs/", ts.contextBrokerURL, url.QueryEscape(ra.ID))
		req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, cbUrl, bytes.NewBuffer(requestBody))
		req.Header.Add("Content-Type", "application/ld+json")

		resp, err = httpClient.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusMultiStatus {
			errMsg := fmt.Sprintf("failed to update road accident in context broker, expected status code %d, but got %d", http.StatusNoContent, resp.StatusCode)
			return errors.New(errMsg)
		}

		logger.Info().Msgf("updated road accident %s in context broker: %s", ra.ID, string(requestBody))

		return nil
	}

	logger.Info().Msgf("published road accident %s to context broker: %s", ra.ID, string(requestBody))

	return err
}

func getLocationFromString(location string) *geojson.GeoJSONProperty {
	position := location[7 : len(location)-1]

	Longitude := strings.Split(position, " ")[0]
	newLong, _ := strconv.ParseFloat(Longitude, 32)
	Latitude := strings.Split(position, " ")[1]
	newLat, _ := strconv.ParseFloat(Latitude, 32)

	return geojson.CreateGeoJSONPropertyFromWGS84(newLong, newLat)
}
