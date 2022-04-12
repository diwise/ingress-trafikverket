package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/geojson"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func publishWeatherStationStatus(ctx context.Context, weatherstation weatherStation, contextBrokerURL string) error {

	position := weatherstation.Geometry.Position
	position = position[7 : len(position)-1]

	Longitude := strings.Split(position, " ")[0]
	newLong, _ := strconv.ParseFloat(Longitude, 32)
	Latitude := strings.Split(position, " ")[1]
	newLat, _ := strconv.ParseFloat(Latitude, 32)

	device := fiware.NewDevice("se:trafikverket:temp:"+weatherstation.ID, fmt.Sprintf("t=%.1f", weatherstation.Measurement.Air.Temp))
	device.Location = geojson.CreateGeoJSONPropertyFromWGS84(newLong, newLat)

	patchBody, err := json.Marshal(device)
	if err != nil {
		return err
	}

	ctx, span := tracer.Start(ctx, "publish-weatherstation")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	url := fmt.Sprintf("%s/ngsi-ld/v1/entities/%s/attrs/", contextBrokerURL, device.ID)

	req, _ := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(patchBody))

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return err
	}

	return nil
}
