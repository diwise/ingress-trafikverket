package weathersvc

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/diwise/context-broker/pkg/ngsild"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types"
	test "github.com/diwise/context-broker/pkg/test"
	. "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
)

func TestWeather(t *testing.T) {
	is, ctxbroker, ws := setupMockWeatherService(t, http.StatusOK, responseJSON)

	_, err := ws.getAndPublishWeatherStations(context.Background(), "")

	is.NoErr(err)
	is.Equal(len(ctxbroker.MergeEntityCalls()), 8)  // should first attempt to merge all weather stations
	is.Equal(len(ctxbroker.CreateEntityCalls()), 8) // create should equal the merge attempts, as each weatherstation is unknown
}

func TestGetWeatherStationStatus(t *testing.T) {
	is, _, ws := setupMockWeatherService(t, http.StatusOK, responseJSON)

	_, err := ws.getWeatherStationStatus(context.Background(), "")

	is.NoErr(err)
}

func TestGetWeatherStationStatusFail(t *testing.T) {
	is, _, ws := setupMockWeatherService(t, http.StatusUnauthorized, "")

	_, err := ws.getWeatherStationStatus(context.Background(), "")

	is.True(err != nil) // Test failed, expected an error but got none
}

func TestPublishWeatherStationStatus(t *testing.T) {
	is, ctxbroker, ws := setupMockWeatherService(t, 0, "")

	weather := weatherStation{
		ID:          "123",
		Name:        "ABC",
		Geometry:    geometry{Position: "POINT (17.345039367675781 62.276519775390625)"},
		Measurement: measurement{Air: air{12.0, 86.5}, MeasureTime: "2020-03-16T08:15:50.156Z"},
	}

	err := ws.publishWeatherStationStatus(context.Background(), weather)
	is.NoErr(err)

	is.Equal(len(ctxbroker.MergeEntityCalls()), 1)  // first attempt to merge
	is.Equal(len(ctxbroker.CreateEntityCalls()), 1) // on failure to merge due to not found error, should create instead

}

func TestPublishWeatherStationConvertsTimeProperly(t *testing.T) {
	is, ctxbroker, ws := setupMockWeatherService(t, 0, "")

	weather := weatherStation{
		ID:          "123",
		Name:        "ABC",
		Geometry:    geometry{Position: "POINT (17.345039367675781 62.276519775390625)"},
		Measurement: measurement{Air: air{12.0, 92.0}, MeasureTime: "2020-03-16T09:10:00.000+01:00"},
	}

	err := ws.publishWeatherStationStatus(context.Background(), weather)
	is.NoErr(err)

	e := ctxbroker.CreateEntityCalls()[0].Entity
	eBytes, _ := e.MarshalJSON()

	dateObserved := `"dateObserved":{"type":"Property","value":{"@type":"DateTime","@value":"2020-03-16T08:10:00Z"}}`

	is.True(strings.Contains(string(eBytes), dateObserved))
}

func setupMockWeatherService(t *testing.T, tfvStatusCode int, tfvBody string) (*is.I, *test.ContextBrokerClientMock, WeatherService) {
	is := is.New(t)
	tfvMock := NewMockServiceThat(
		Expects(is, expects.AnyInput()),
		Returns(
			response.Code(tfvStatusCode),
			response.Body([]byte(tfvBody)),
		),
	)

	ctxBroker := &test.ContextBrokerClientMock{
		CreateEntityFunc: func(ctx context.Context, entity types.Entity, headers map[string][]string) (*ngsild.CreateEntityResult, error) {
			return nil, nil
		},
		MergeEntityFunc: func(ctx context.Context, entityID string, fragment types.EntityFragment, headers map[string][]string) (*ngsild.MergeEntityResult, error) {
			return nil, ngsierrors.ErrNotFound
		},
	}
	ws := NewWeatherService(context.Background(), "", tfvMock.URL(), ctxBroker)

	return is, ctxBroker, ws
}

const responseJSON string = "{ \"RESPONSE\":{\"RESULT\":[{\"WeatherStation\":[{\"Active\":true, \"Geometry\":{\"WGS84\":\"POINT (17.31352996826172 62.470909118652344)\"},\"Id\":\"SE_STA_VVIS2216\", \"Measurement\":{\"MeasureTime\":\"2023-01-09T14:10:01.000+01:00\", \"Air\":{\"Temp\":1.4,\"RelativeHumidity\":98.8}},\"ModifiedTime\":\"2023-01-09T13:17:04.210Z\",\"Name\":\"Timrå\"},{\"Active\":true, \"Geometry\":{\"WGS84\":\"POINT (17.452890396118164 62.51641082763672)\"},\"Id\":\"SE_STA_VVIS2221\", \"Measurement\":{\"MeasureTime\":\"2023-01-09T14:10:01.000+01:00\", \"Precipitation\":{\"Amount\":0}, \"Air\":{\"Temp\":1.5,\"RelativeHumidity\":98.8}, \"Wind\":{\"Direction\":90,\"Force\":1.1,\"ForceMax\":2.7}},\"ModifiedTime\":\"2023-01-09T13:17:04.386Z\",\"Name\":\"Deltavägen\"},{\"Active\":true, \"Geometry\":{\"WGS84\":\"POINT (15.851380348205566 62.55556869506836)\"},\"Id\":\"SE_STA_VVIS2226\", \"Measurement\":{\"MeasureTime\":\"2023-01-09T14:10:01.000+01:00\", \"Air\":{\"Temp\":1.2,\"RelativeHumidity\":94.8}, \"Wind\":{\"Direction\":135,\"Force\":0.7,\"ForceMax\":3}},\"ModifiedTime\":\"2023-01-09T13:17:04.528Z\",\"Name\":\"Tälje\"},{\"Active\":true, \"Geometry\":{\"WGS84\":\"POINT (16.366090774536133 62.473670959472656)\"},\"Id\":\"SE_STA_VVIS2241\", \"Measurement\":{\"MeasureTime\":\"2023-01-09T14:10:01.000+01:00\", \"Air\":{\"Temp\":0.6,\"RelativeHumidity\":98.9}, \"Wind\":{\"Direction\":135,\"Force\":1.3,\"ForceMax\":3}},\"ModifiedTime\":\"2023-01-09T13:17:04.904Z\",\"Name\":\"Torpshammar\"},{\"Active\":true, \"Geometry\":{\"WGS84\":\"POINT (17.34100914001465 62.38862991333008)\"},\"Id\":\"SE_STA_VVIS2244\", \"Measurement\":{\"MeasureTime\":\"2023-01-09T14:10:00.000+01:00\", \"Precipitation\":{\"Amount\":0}, \"Air\":{\"Temp\":1.6,\"RelativeHumidity\":98.9}, \"Wind\":{\"Direction\":135,\"Force\":0,\"ForceMax\":0}},\"ModifiedTime\":\"2023-01-09T13:17:04.998Z\",\"Name\":\"Sundsvall 2\"},{\"Active\":true, \"Geometry\":{\"WGS84\":\"POINT (17.34503936767578 62.276519775390625)\"},\"Id\":\"SE_STA_VVIS2245\", \"Measurement\":{\"MeasureTime\":\"2023-01-09T14:10:01.000+01:00\", \"Precipitation\":{\"Amount\":0.2}, \"Air\":{\"Temp\":1.5,\"RelativeHumidity\":98.8}, \"Wind\":{\"Direction\":90,\"Force\":0.7,\"ForceMax\":2.8}},\"ModifiedTime\":\"2023-01-09T13:17:05.014Z\",\"Name\":\"Ljungan\"},{\"Active\":true, \"Geometry\":{\"WGS84\":\"POINT (17.313539505004883 62.47090148925781)\"},\"Id\":\"SE_STA_VVIS2216100\", \"Measurement\":{\"MeasureTime\":\"2023-01-09T14:10:01.000+01:00\", \"Air\":{\"Temp\":1.4,\"RelativeHumidity\":98.8}},\"ModifiedTime\":\"2023-01-09T13:17:10.679Z\",\"Name\":\"2216 Timrå Fjärryta\"},{\"Active\":true, \"Geometry\":{\"WGS84\":\"POINT (17.452899932861328 62.51641082763672)\"},\"Id\":\"SE_STA_VVIS2221100\", \"Measurement\":{\"MeasureTime\":\"2023-01-09T14:10:01.000+01:00\", \"Precipitation\":{\"Amount\":0}, \"Air\":{\"Temp\":1.5,\"RelativeHumidity\":98.8}},\"ModifiedTime\":\"2023-01-09T13:17:10.679Z\",\"Name\":\"2221 Deltavägen Fjärryta\"}], \"INFO\":{\"LASTCHANGEID\":\"7186640915220399997\"}}]}}"
