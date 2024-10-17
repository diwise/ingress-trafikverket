package roadaccidents

import (
	"context"
	"net/http"
	"testing"

	"github.com/diwise/context-broker/pkg/ngsild"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	test "github.com/diwise/context-broker/pkg/test"
	. "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
)

func TestRetrievingRoadAccidentsFromTFV(t *testing.T) {
	is, _, ts, ms := setupMockRoadAccident(t, http.StatusOK, tfvResponseJSON)
	defer ms.Close()

	_, err := ts.getRoadAccidentsFromTFV(context.Background(), "")
	is.NoErr(err)
}

func TestPublishingRoadAccidentsToContextBroker(t *testing.T) {
	is, cb, ts, ms := setupMockRoadAccident(t, 0, "")
	defer ms.Close()

	dev := tfvDeviation{
		Id:     "id",
		IconId: "roadAccident",
		Geometry: tfvGeometry{
			Point: tfvPoint{
				WGS84: "POINT (13.0958767 55.9722252)",
			},
		},
		Message:   "this is not a drill",
		StartTime: "2022-04-21T19:37:57.000+02:00",
		EndTime:   "2022-04-21T20:45:00.000+02:00",
	}

	_ = ts.publishRoadAccidentToContextBroker(context.Background(), dev, false)

	is.Equal(len(cb.MergeEntityCalls()), 1)
	is.Equal(cb.MergeEntityCalls()[0].EntityID, "urn:ngsi-ld:RoadAccident:id")
	is.NoErr(entities.ValidateFragmentAttributes(
		cb.MergeEntityCalls()[0].Fragment,
		map[string]any{
			"dateCreated": "2022-04-21T17:37:57Z",
			"description": "this is not a drill",
		},
	))
}

func TestThatLastChangeIDStoresCorrectly(t *testing.T) {
	is, _, ts, ms := setupMockRoadAccident(t, http.StatusOK, tfvResponseJSON)
	defer ms.Close()

	lastChangeID, err := ts.getAndPublishRoadAccidents(context.Background(), "0")
	is.NoErr(err)
	is.Equal(lastChangeID, "7426311386101186961")
}

func TestThatIfSituationIsDeletedStatusAttributeChanges(t *testing.T) {
	is, cb, ts, ms := setupMockRoadAccident(t, http.StatusOK, tfvResponseJSON)
	defer ms.Close()

	_, err := ts.getAndPublishRoadAccidents(context.Background(), "0")
	is.NoErr(err)

	dev := tfvDeviation{
		Id:     "SE_STA_TRISSID_1_6923722",
		IconId: "roadAccident",
		Geometry: tfvGeometry{
			Point: tfvPoint{
				WGS84: "POINT (13.0958767 55.9722252)",
			},
		},
		StartTime: "2022-04-21T19:37:57.000+02:00",
		EndTime:   "2022-04-21T20:45:00.000+02:00",
	}

	const IsDeleted bool = true
	err = ts.publishRoadAccidentToContextBroker(context.Background(), dev, IsDeleted)

	is.NoErr(err)
	is.Equal(len(cb.MergeEntityCalls()), 2) // this is 2 because the first publishing of a road accident will also initially trigger the mergeentity function, before moving on to create
	is.NoErr(entities.ValidateFragmentAttributes(
		cb.MergeEntityCalls()[1].Fragment,
		map[string]any{"status": "solved"},
	))
}

func setupMockRoadAccident(t *testing.T, tfvCode int, tfvBody string) (*is.I, *test.ContextBrokerClientMock, *roadAccidentSvc, MockService) {
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

	ts := NewService(context.Background(), "", tfvMock.URL(), "0", ctxBroker)

	return is, ctxBroker, ts.(*roadAccidentSvc), tfvMock
}

const tfvResponseJSON string = `{ "RESPONSE":{"RESULT":[{"Situation":[{"Deleted":true,"Deviation":[{"EndTime":"2024-10-16T12:30:00.000+02:00", "Geometry":{ "Point":{"WGS84":"POINT (18.4573116 63.2837563)"}},"IconId":"roadAccident","Id":"SE_STA_TRISSID_1_6923722","Message":"Olycka med flera fordon i höjd med Långsvedjan. Vägen är avstängd under räddningsarbetet.","StartTime":"2024-10-16T10:51:28.000+02:00"},{"EndTime":"2024-10-16T12:30:00.000+02:00", "Geometry":{ "Point":{"WGS84":"POINT (18.4573116 63.2837563)"}},"IconId":"roadClosed","Id":"SE_STA_TRISSID_2_6923722","StartTime":"2024-10-16T10:51:28.000+02:00"}]}], "INFO":{"LASTCHANGEID":"7426311386101186961"}}]}}`
