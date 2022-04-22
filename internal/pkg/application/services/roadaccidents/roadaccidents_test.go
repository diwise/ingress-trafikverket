package roadaccidents

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
)

func TestRetrievingRoadAccidentsFromTFV(t *testing.T) {
	is, ts := setupMockRoadAccident(t, http.StatusOK, tfvResponseJSON, 0, "")

	_, err := ts.getRoadAccidentsFromTFV(context.Background(), "")
	is.NoErr(err)
}

func TestPublishingRoadAccidentsToContextBroker(t *testing.T) {
	is, ts := setupMockRoadAccident(t, 0, "", http.StatusCreated, "")

	dev := tfvDeviation{
		Id:     "id",
		IconId: "roadAccident",
		Geometry: tfvGeometry{
			"POINT (13.0958767 55.9722252)",
		},
		StartTime: "2022-04-21T19:37:57.000+02:00",
		EndTime:   "2022-04-21T20:45:00.000+02:00",
	}

	err := ts.publishRoadAccidentsToContextBroker(context.Background(), dev)
	is.NoErr(err)
}

func TestThatLastChangeIDStoresCorrectly(t *testing.T) {
	is, ts := setupMockRoadAccident(t, http.StatusOK, tfvResponseJSON, http.StatusCreated, roadAccidentJSON)

	lastChangeID, err := ts.getAndPublishRoadAccidents(context.Background(), "0")
	is.NoErr(err)
	is.Equal(lastChangeID, "7089127599774892692")
}

func TestThatIfSituationIsDeletedItTriggersUpdateStatus(t *testing.T) {
	is, ts := setupMockRoadAccident(t, http.StatusOK, deletedTfvJSON, http.StatusCreated, "")

	_, err := ts.getAndPublishRoadAccidents(context.Background(), "0")
	is.NoErr(err)
	// check if "updateRoadAccident" is called, and if status changed
}

func setupMockRoadAccident(t *testing.T, tfvCode int, tfvBody string, ctxCode int, ctxBody string) (*is.I, RoadAccidentSvc) {
	is := is.New(t)
	svcMock := setupMockServiceThatReturns(tfvCode, tfvBody)
	ctxMock := setupMockServiceThatReturns(ctxCode, ctxBody)
	ts := NewService("", svcMock.URL, ctxMock.URL)

	return is, ts
}

func setupMockServiceThatReturns(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}

const roadAccidentJSON string = `{"id":"urn:ngsi-ld:RoadAccident:SE_STA_TRISSID_1_16279394","type":"RoadAccident","@context":["https://raw.githubusercontent.com/smart-data-models/dataModel.Transportation/master/context.jsonld\"],"accidentDate":{"type":"Property","value":{"@type":"DateTime","@value":"2022-04-21T19:37:57.000+02:00"}},"location":{"type":"GeoProperty","value":{"type":"Point","coordinates":[12.220904350280762,57.68485641479492]}},"description",:{"type":"Property","value":"Trafikolycka - singel."}"status":{"type":"Property","value":"onGoing"}}`

const tfvResponseJSON string = `{"RESPONSE":{"RESULT":[{"Situation":[{"Deleted":false,"Deviation":[{"EndTime":"2022-04-21T21:15:00.000+02:00","Geometry":{"WGS84":"POINT (13.0958767 55.9722252)"},"IconId":"roadAccident","Id":"SE_STA_TRISSID_1_9879392","Message":"Trafikolycka med flera fordon söder om Kågeröd.","StartTime":"2022-04-21T20:12:01.000+02:00"}]}],"INFO":{"LASTCHANGEID":"7089127599774892692"}}]}}`

const deletedTfvJSON string = `{"RESPONSE":{"RESULT":[{"Situation":[{"Deleted":true,"Deviation":[{"EndTime":"2022-04-21T21:15:00.000+02:00","Geometry":{"WGS84":"POINT (13.0958767 55.9722252)"},"IconId":"roadAccident","Id":"SE_STA_TRISSID_1_9879392","Message":"Trafikolycka med flera fordon söder om Kågeröd.","StartTime":"2022-04-21T20:12:01.000+02:00"}]}],"INFO":{"LASTCHANGEID":"7089127597489431414"}}]}}`
