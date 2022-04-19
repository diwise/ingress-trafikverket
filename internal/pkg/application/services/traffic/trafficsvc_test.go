package trafficsvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestTrafficFromTFV(t *testing.T) {
	is, ts := setupMockTrafficService(t, http.StatusOK, tfvResponseJSON)

	_, err := ts.getTrafficInformationFromTFV(context.Background())
	is.NoErr(err)
}

func TestTrafficFromSDL(t *testing.T) {
	is, ts := setupMockTrafficService(t, http.StatusOK, sdlResponseJSON)

	_, err := ts.getTrafficInformationFromSDL(context.Background())
	is.NoErr(err)
}

func setupMockTrafficService(t *testing.T, statusCode int, body string) (*is.I, TrafficService) {
	is := is.New(t)
	svcMock := setupMockServiceThatReturns(statusCode, body)
	ts := NewTrafficService(zerolog.Logger{}, "", svcMock.URL)

	return is, ts
}

func setupMockServiceThatReturns(statusCode int, body string) *httptest.Server {

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}

const tfvResponseJSON string = `{"RESPONSE":{"RESULT":[{"Situation":[{"Deviation":[{"Geometry":{"WGS84":"POINT (13.684638 59.5861626)"},"IconId":"roadAccident","Id":"xxxxxxxx"}]},{"Deviation":[{"Geometry":{"WGS84":"POINT (14.8251829 59.2525826)"},"IconId":"roadAccident","Id":"xxxxxxxxxx"}]},{"Deviation":[{"Geometry":{"WGS84":"POINT (15.9879961 59.51102)"},"IconId":"roadAccident","Id":"xxxxxxxxxx"}]}]}]}}`

const sdlResponseJSON string = `{"type":"FeatureCollection","name":"Sundsvall Växer trafikstörningar","crs":{"type":"name","properties":{"name":"urn:ogc:def:crs:EPSG::3006"}},"features":[{"type":"Feature","geometry":{"type":"GeometryCollection",geometries":[{"type":"Point","coordinates":[623408.2126134963,6906241.8869982]},{"type":"Polygon","coordinates":[[[623416.3165564446,6906254.792244375],]]}]},"properties":{"title":"Mjösundsvägen","description":"","restrictions":"","level":"LARGE","start": "2018-08-31","end": "2022-01-31"}},{"type":"Feature","geometry":{"type":"GeometryCollection","geometries":[{"type":"Point","coordinates":[619122.4199999999,6918869.4925]}]},"properties":{"title":"Gränsgatan","description":"","level":"SMALL","start":"2021-11-30","end": "2021-12-10"}}]}`
