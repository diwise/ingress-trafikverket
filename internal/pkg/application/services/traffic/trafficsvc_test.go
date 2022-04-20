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
