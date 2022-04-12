package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
)

func TestMain(m *testing.M) {

	os.Exit(m.Run())
}

func TestWeather(t *testing.T) {
	is := is.New(t)
	tfvMock := setupMockServiceThatReturns(http.StatusOK, responseJSON)
	ctxBrokerMock := setupMockServiceThatReturns(http.StatusNoContent, "")

	_, err := getAndPublishWeatherStationStatus(context.Background(), zerolog.Logger{}, "", "", tfvMock.URL, ctxBrokerMock.URL)
	is.NoErr(err)
}

func TestGetWeatherStationStatus(t *testing.T) {
	is := is.New(t)
	mockService := setupMockServiceThatReturns(http.StatusOK, responseJSON)

	_, err := getWeatherStationStatus(context.Background(), mockService.URL, "", "")

	is.NoErr(err)
}

func TestGetWeatherStationStatusFail(t *testing.T) {
	is := is.New(t)
	mockService := setupMockServiceThatReturns(http.StatusUnauthorized, "")

	_, err := getWeatherStationStatus(context.Background(), mockService.URL, "", "")

	is.True(err != nil) // Test failed, expected an error but got none
}

func TestPublishWeatherStationStatus(t *testing.T) {
	is := is.New(t)
	mockService := setupMockServiceThatReturns(http.StatusNoContent, "")

	weather := weatherStation{
		ID:          "123",
		Name:        "ABC",
		Geometry:    geometry{Position: "17.345039367675781 62.276519775390625"},
		Measurement: measurement{Air: air{12.0}, MeasureTime: "2020-03-16T08:15:50.156Z"},
	}

	err := publishWeatherStationStatus(context.Background(), weather, mockService.URL)

	is.NoErr(err)
}

func setupMockServiceThatReturns(statusCode int, body string) *httptest.Server {

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}

const responseJSON string = "{ \"RESPONSE\":{\"RESULT\":[{\"WeatherStation\":[{ \"Geometry\":{\"WGS84\":\"POINT (17.047550201416016 62.362770080566406)\"},\"Id\":\"SE_STA_VVIS2213\", \"Measurement\":{\"MeasureTime\":\"2020-03-16T09:10:00.000+01:00\", \"Air\":{\"Temp\":0.9}},\"ModifiedTime\":\"2020-03-16T08:15:49.889Z\",\"Name\":\"Vattjom\"},{ \"Geometry\":{\"WGS84\":\"POINT (17.122060775756836 62.552879333496094)\"},\"Id\":\"SE_STA_VVIS2214\", \"Measurement\":{\"MeasureTime\":\"2020-03-16T09:10:00.000+01:00\", \"Air\":{\"Temp\":1}},\"ModifiedTime\":\"2020-03-16T08:15:49.904Z\",\"Name\":\"Kävstabron\"},{ \"Geometry\":{\"WGS84\":\"POINT (17.313529968261719 62.470909118652344)\"},\"Id\":\"SE_STA_VVIS2216\", \"Measurement\":{\"MeasureTime\":\"2020-03-16T09:10:02.000+01:00\", \"Air\":{\"Temp\":2.6}},\"ModifiedTime\":\"2020-03-16T08:15:49.904Z\",\"Name\":\"Timrå\"},{ \"Geometry\":{\"WGS84\":\"POINT (17.452890396118164 62.516410827636719)\"},\"Id\":\"SE_STA_VVIS2221\", \"Measurement\":{\"MeasureTime\":\"2020-03-16T09:10:00.000+01:00\", \"Air\":{\"Temp\":1.9}},\"ModifiedTime\":\"2020-03-16T08:15:49.997Z\",\"Name\":\"Deltavägen\"},{ \"Geometry\":{\"WGS84\":\"POINT (15.851380348205566 62.555568695068359)\"},\"Id\":\"SE_STA_VVIS2226\", \"Measurement\":{\"MeasureTime\":\"2020-03-16T09:10:00.000+01:00\", \"Air\":{\"Temp\":1.7}},\"ModifiedTime\":\"2020-03-16T08:15:50.044Z\",\"Name\":\"Tälje\"},{ \"Geometry\":{\"WGS84\":\"POINT (16.366090774536133 62.473670959472656)\"},\"Id\":\"SE_STA_VVIS2241\", \"Measurement\":{\"MeasureTime\":\"2020-03-16T09:10:00.000+01:00\", \"Air\":{\"Temp\":1.2}},\"ModifiedTime\":\"2020-03-16T08:15:50.124Z\",\"Name\":\"Torpshammar\"},{ \"Geometry\":{\"WGS84\":\"POINT (17.341009140014648 62.388629913330078)\"},\"Id\":\"SE_STA_VVIS2244\", \"Measurement\":{\"MeasureTime\":\"2020-03-16T09:10:02.000+01:00\", \"Air\":{\"Temp\":1.5}},\"ModifiedTime\":\"2020-03-16T08:15:50.140Z\",\"Name\":\"Sundsvall 2\"},{ \"Geometry\":{\"WGS84\":\"POINT (17.345039367675781 62.276519775390625)\"},\"Id\":\"SE_STA_VVIS2245\", \"Measurement\":{\"MeasureTime\":\"2020-03-16T09:10:01.000+01:00\", \"Air\":{\"Temp\":2}},\"ModifiedTime\":\"2020-03-16T08:15:50.156Z\",\"Name\":\"Ljungan\"}],\"INFO\":{\"LASTCHANGEID\":\"\"}}]}}"
