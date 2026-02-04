package weathersvc

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	test "github.com/diwise/context-broker/pkg/test"
	httptest "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
)

func TestWeather(t *testing.T) {
	is, ctxbroker, ws, ms := setupMockWeatherService(t, http.StatusOK, responseJSON)
	defer ms.Close()

	_, err := ws.getAndPublishWeatherMeasurepoints(context.Background(), "")

	is.NoErr(err)
	is.Equal(len(ctxbroker.MergeEntityCalls()), 19)  // should first attempt to merge all weather stations
	is.Equal(len(ctxbroker.CreateEntityCalls()), 19) // create should equal the merge attempts, as each weathermeasurepoint is unknown
}

func TestGetWeatherMeasurepointStatus(t *testing.T) {
	is, _, ws, ms := setupMockWeatherService(t, http.StatusOK, responseJSON)
	defer ms.Close()

	_, err := ws.getWeatherMeasurepointStatus(context.Background(), "")

	is.NoErr(err)
}

func TestGetWeatherMeasurepointStatusFail(t *testing.T) {
	is, cb, ws, ms := setupMockWeatherService(t, http.StatusUnauthorized, "")
	defer ms.Close()

	_, err := ws.getWeatherMeasurepointStatus(context.Background(), "")

	is.True(err != nil) // Test failed, expected an error but got none
	is.Equal(len(cb.MergeEntityCalls()), 0)
}

func TestPublishWeatherMeasurepointStatus(t *testing.T) {
	is, ctxbroker, ws, ms := setupMockWeatherService(t, 0, "")
	defer ms.Close()

	tm, _ := time.Parse(time.RFC3339, "2020-03-16T08:15:50.156Z")

	weather := weatherMeasurepoint{
		ID:           "123",
		Name:         "ABC",
		Geometry:     geometry{Position: "POINT (17.345039367675781 62.276519775390625)"},
		ModifiedTime: tm,
		Observation: observation{
			Air: &air{
				Temperature:      osv{"", "", 12.0},
				RelativeHumidity: osv{"", "", 86.5},
			},
		},
	}

	_ = ws.publishWeatherMeasurepointStatus(context.Background(), weather)

	is.Equal(len(ctxbroker.MergeEntityCalls()), 1) // assert that we have a call to apply expectations on
	is.NoErr(entities.ValidateFragmentAttributes(
		ctxbroker.MergeEntityCalls()[0].Fragment,
		map[string]any{"name": "ABC", "temperature": 12.0, "humidity": 0.865},
	))
}

func TestPublishWeatherMeasurepointConvertsTimeProperly(t *testing.T) {
	is, ctxbroker, ws, ms := setupMockWeatherService(t, 0, "")
	defer ms.Close()

	tm, _ := time.Parse(time.RFC3339, "2020-03-16T09:10:00.000+01:00")

	weather := weatherMeasurepoint{
		ID:           "123",
		Name:         "ABC",
		Geometry:     geometry{Position: "POINT (17.345039367675781 62.276519775390625)"},
		ModifiedTime: tm.UTC(),
		Observation: observation{
			Air: &air{
				Temperature:      osv{"", "", 12.0},
				RelativeHumidity: osv{"", "", 92.0},
			},
		},
	}

	_ = ws.publishWeatherMeasurepointStatus(context.Background(), weather)

	is.NoErr(entities.ValidateFragmentAttributes(
		ctxbroker.CreateEntityCalls()[0].Entity,
		map[string]any{"dateObserved": "2020-03-16T08:10:00Z"},
	))
}

func setupMockWeatherService(t *testing.T, tfvStatusCode int, tfvBody string) (*is.I, *test.ContextBrokerClientMock, *weatherSvc, httptest.MockService) {
	is := is.New(t)
	tfvMock := httptest.NewMockServiceThat(
		httptest.Expects(is, expects.AnyInput()),
		httptest.Returns(
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

	ws := NewWeatherService(context.Background(), "", tfvMock.URL(), "", ctxBroker)

	return is, ctxBroker, ws.(*weatherSvc), tfvMock
}

const responseJSON string = `{ "RESPONSE":{"RESULT":[{"WeatherMeasurepoint":[{"Id":"2202","Name":"Råsta", "Geometry":{"WGS84":"POINT (17.34482 62.43064)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":2.8}, "RelativeHumidity":{"Value":91}},"Wind":[{ "Direction":{"Value":155}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:41:47.131Z"},{"Id":"2212","Name":"Nedansjö", "Geometry":{"WGS84":"POINT (16.87648 62.37616)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":0.8}, "RelativeHumidity":{"Value":98.3}},"Wind":[{ "Direction":{"Value":16}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:41:47.134Z"},{"Id":"2213","Name":"Vattjom", "Geometry":{"WGS84":"POINT (17.04746 62.36276)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":0.9}, "RelativeHumidity":{"Value":97.4}},"Wind":[{ "Direction":{"Value":211}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:41:47.137Z"},{"Id":"2214","Name":"Kävstabron", "Geometry":{"WGS84":"POINT (17.12191 62.55283)"}, "Observation":{"Sample":"2024-10-16T22:40:03.000+02:00", "Air":{ "Temperature":{"Value":0.3}, "RelativeHumidity":{"Value":98.6}},"Wind":[{ "Direction":{"Value":215}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:41:47.140Z"},{"Id":"2214100","Name":"2214 Kävstabron Fjärryta", "Geometry":{"WGS84":"POINT (17.12206 62.55289)"}, "Observation":{"Sample":"2024-10-16T22:40:03.000+02:00", "Air":{},"Wind":[{}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:41:47.184Z"},{"Id":"2115","Name":"Gryttjesjön", "Geometry":{"WGS84":"POINT (17.27754 62.0778)"}, "Observation":{"Sample":"2024-10-16T22:40:03.000+02:00", "Air":{ "Temperature":{"Value":2.6}, "RelativeHumidity":{"Value":94.6}},"Wind":[{ "Direction":{"Value":233}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:23.043Z"},{"Id":"2128","Name":"Norrhög", "Geometry":{"WGS84":"POINT (15.67114 62.26322)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":3.3}, "RelativeHumidity":{"Value":89}},"Wind":[{ "Direction":{"Value":163}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:23.068Z"},{"Id":"2129","Name":"Furuberg", "Geometry":{"WGS84":"POINT (16.56565 62.07392)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":4.2}, "RelativeHumidity":{"Value":86.5}},"Wind":[{ "Direction":{"Value":231}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:23.071Z"},{"Id":"2201","Name":"Armsjön", "Geometry":{"WGS84":"POINT (17.36961 62.19534)"}, "Observation":{"Sample":"2024-10-16T22:40:03.000+02:00", "Air":{ "Temperature":{"Value":4.9}, "RelativeHumidity":{"Value":87.6}},"Wind":[{ "Direction":{"Value":125}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:23.094Z"},{"Id":"2204","Name":"Högsnäs", "Geometry":{"WGS84":"POINT (17.70954 62.56162)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":1.7}, "RelativeHumidity":{"Value":98.5}},"Wind":[{ "Direction":{"Value":101}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:23.096Z"},{"Id":"2216","Name":"Timrå", "Geometry":{"WGS84":"POINT (17.3137 62.47091)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":-1.1}, "RelativeHumidity":{"Value":98.6}},"Wind":[{ "Direction":{"Value":255}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:23.103Z"},{"Id":"2221","Name":"Deltavägen", "Geometry":{"WGS84":"POINT (17.45301 62.51646)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":0.6}, "RelativeHumidity":{"Value":99.8}},"Wind":[{ "Direction":{"Value":332}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:23.127Z"},{"Id":"2226","Name":"Tälje", "Geometry":{"WGS84":"POINT (15.85131 62.55525)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":-1.1}, "RelativeHumidity":{"Value":99}},"Wind":[{ "Direction":{"Value":314}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:23.138Z"},{"Id":"2241","Name":"Torpshammar", "Geometry":{"WGS84":"POINT (16.36673 62.4734)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":-1.5}, "RelativeHumidity":{"Value":99.7}},"Wind":[{ "Direction":{"Value":87}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:23.168Z"},{"Id":"2245","Name":"Ljungan", "Geometry":{"WGS84":"POINT (17.34481 62.27648)"}, "Observation":{"Sample":"2024-10-16T22:40:03.004+02:00", "Air":{ "Temperature":{"Value":1.3}, "RelativeHumidity":{"Value":98.9}},"Wind":[{ "Direction":{"Value":124}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:23.175Z"},{"Id":"2216100","Name":"2216 Timrå Fjärryta", "Geometry":{"WGS84":"POINT (17.31354 62.4709)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{},"Wind":[{}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:27.928Z"},{"Id":"2221100","Name":"2221 Deltavägen Fjärryta", "Geometry":{"WGS84":"POINT (17.4529 62.51641)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{},"Wind":[{}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:43:27.932Z"},{"Id":"2244","Name":"Sundsvall 2", "Geometry":{"WGS84":"POINT (17.34096 62.38865)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":5.3}, "RelativeHumidity":{"Value":80.3}},"Wind":[{ "Direction":{"Value":237}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:44:48.593Z"},{"Id":"7601","Name":"Ånge GBG", "Geometry":{"WGS84":"POINT (15.64424 62.52549)"}, "Observation":{"Sample":"2024-10-16T22:40:03.001+02:00", "Air":{ "Temperature":{"Value":5.6}, "RelativeHumidity":{"Value":74.5}},"Wind":[{ "Direction":{"Value":297}}]},"Deleted":false,"ModifiedTime":"2024-10-16T20:44:48.687Z"}], "INFO":{"LASTCHANGEID":"7426477292097896709"}}]}}`
