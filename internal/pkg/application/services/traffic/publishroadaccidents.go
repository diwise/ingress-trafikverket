package trafficsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/diwise/ingress-trafikverket/internal/pkg/fiware"
	"github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/geojson"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func (ts *ts) publishRoadAccidentsToContextBroker(resp []byte, ctx context.Context) error {
	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	if resp == nil {
		ts.log.Info().Msg("no new incidents to send to context broker")
		return nil
	}

	tfvResp := tfvResponse{}

	err := json.Unmarshal(resp, &tfvResp)
	if err != nil {
		return err
	}

	for _, dev := range tfvResp.Response.Result[0].Situation[0].Deviation {
		ra := fiware.NewRoadAccident(dev.Id)
		ra.AccidentDate = *ngsitypes.CreateDateTimeProperty(dev.StartTime)
		ra.Location = getLocationFromString(dev.Geometry.WGS84)
		//probably add checks to see if a property is empty

		requestBody, err := json.Marshal(ra)
		if err != nil {
			return err
		}

		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.contextBrokerURL, bytes.NewBuffer(requestBody))

		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusCreated {
			log.Error().Msgf("failed to send RoadAccident to context broker, expected status code %d, but got %d", http.StatusOK, resp.StatusCode)
			return errors.New("")
		}
	}

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
