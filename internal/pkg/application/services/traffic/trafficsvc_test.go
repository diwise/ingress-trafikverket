package trafficsvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestTraffic(t *testing.T) {
	is, ts := setupMockTrafficService(t, http.StatusOK, responseJSON)

	_, err := ts.getTrafficInformation(context.Background())
	is.NoErr(err)
}

func setupMockTrafficService(t *testing.T, tfvStatusCode int, tfvBody string) (*is.I, TrafficService) {
	is := is.New(t)
	tfvMock := setupMockServiceThatReturns(tfvStatusCode, tfvBody)
	ws := NewTrafficService(zerolog.Logger{}, "", tfvMock.URL)

	return is, ws
}

func setupMockServiceThatReturns(statusCode int, body string) *httptest.Server {

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}

const responseJSON string = `{"RESPONSE":{"RESULT":[{"Situation":[{"Deviation":[{"Header":"Ängöleden","IconId":"ferryServiceNotOperating","Message":"Färjan inställd på grund underhållsarbete 2022-04-12 mellan klockan  09:05-16:05.","MessageCode":"Färja","MessageType":"Färjor"}]}]}]}}"}`
