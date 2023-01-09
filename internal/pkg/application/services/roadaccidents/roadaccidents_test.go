package roadaccidents

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

func TestRetrievingRoadAccidentsFromTFV(t *testing.T) {
	is, cb, ts := setupMockRoadAccident(t, http.StatusOK, tfvResponseJSON)

	_, err := ts.getRoadAccidentsFromTFV(context.Background(), "")
	is.NoErr(err)
	is.Equal(len(cb.MergeEntityCalls()), 0)
}

func TestPublishingRoadAccidentsToContextBroker(t *testing.T) {
	is, cb, ts := setupMockRoadAccident(t, 0, "")

	dev := tfvDeviation{
		Id:     "id",
		IconId: "roadAccident",
		Geometry: tfvGeometry{
			"POINT (13.0958767 55.9722252)",
		},
		StartTime: "2022-04-21T19:37:57.000+02:00",
		EndTime:   "2022-04-21T20:45:00.000+02:00",
	}

	err := ts.publishRoadAccidentToContextBroker(context.Background(), dev, false)
	is.NoErr(err)
	is.Equal(len(cb.MergeEntityCalls()), 1)
}

func TestThatWeSkipPublishingRoadAccidentsWithPreviousIds(t *testing.T) {
	is, _, ts := setupMockRoadAccident(t, http.StatusOK, tfvResponseJSON)

	lastChangeID, err := ts.getAndPublishRoadAccidents(context.Background(), "0")
	is.NoErr(err)

	_, err = ts.getAndPublishRoadAccidents(context.Background(), lastChangeID)
	is.NoErr(err)
}

func TestThatLastChangeIDStoresCorrectly(t *testing.T) {
	is, _, ts := setupMockRoadAccident(t, http.StatusOK, tfvResponseJSON)

	lastChangeID, err := ts.getAndPublishRoadAccidents(context.Background(), "0")
	is.NoErr(err)
	is.Equal(lastChangeID, "7089127599774892692")
}

func TestThatIfSituationIsDeletedStatusAttributeChanges(t *testing.T) {
	is, cb, ts := setupMockRoadAccident(t, http.StatusOK, tfvResponseJSON)

	_, err := ts.getAndPublishRoadAccidents(context.Background(), "0")
	is.NoErr(err)
	is.Equal(len(cb.CreateEntityCalls()), 1)

	dev := tfvDeviation{
		Id:     "SE_STA_TRISSID_1_9879392",
		IconId: "roadAccident",
		Geometry: tfvGeometry{
			"POINT (13.0958767 55.9722252)",
		},
		StartTime: "2022-04-21T19:37:57.000+02:00",
		EndTime:   "2022-04-21T20:45:00.000+02:00",
	}
	err = ts.publishRoadAccidentToContextBroker(context.Background(), dev, true)
	is.NoErr(err)
	is.Equal(len(cb.MergeEntityCalls()), 2) // this is 2 because the first publishing of a road accident will also initially trigger the mergeentity function, before moving on to create

	e := cb.MergeEntityCalls()[1] // the second merge entity is the one containing status solved
	eBytes, err := e.Fragment.MarshalJSON()
	is.NoErr(err)

	deleted := `"status":{"type":"Property","value":"solved"}`
	is.True(strings.Contains(string(eBytes), deleted))
}

func setupMockRoadAccident(t *testing.T, tfvCode int, tfvBody string) (*is.I, *test.ContextBrokerClientMock, RoadAccidentSvc) {
	is := is.New(t)
	tfvMock := NewMockServiceThat(
		Expects(is, expects.AnyInput()),
		Returns(
			response.Code(tfvCode),
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
	ts := NewService("", tfvMock.URL(), "0", ctxBroker)

	return is, ctxBroker, ts
}

const tfvResponseJSON string = `{"RESPONSE":{"RESULT":[{"Situation":[{"Deleted":false,"Deviation":[{"EndTime":"2022-04-21T21:15:00.000+02:00","Geometry":{"WGS84":"POINT (13.0958767 55.9722252)"},"IconId":"roadAccident","Id":"SE_STA_TRISSID_1_9879392","Message":"Trafikolycka med flera fordon söder om Kågeröd.","StartTime":"2022-04-21T20:12:01.000+02:00"}]}],"INFO":{"LASTCHANGEID":"7089127599774892692"}}]}}`
