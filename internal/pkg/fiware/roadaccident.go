package fiware

import (
	"github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/geojson"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
)

type RoadAccident struct {
	ngsitypes.BaseEntity
	AccidentDate ngsitypes.DateTimeProperty `json:"accidentDate,omitempty"`
	Location     *geojson.GeoJSONProperty   `json:"location,omitempty"`
	Description  ngsitypes.TextProperty     `json:"description,omitempty"`
	DateCreated  ngsitypes.DateTimeProperty `json:"dateCreated,omitempty"`
	DateModified ngsitypes.DateTimeProperty `json:"dateModified,omitempty"`
	Status       ngsitypes.DateTimeProperty `json:"status,omitempty"`
}

func NewRoadAccident(entityID string) RoadAccident {
	ra := RoadAccident{
		BaseEntity: ngsitypes.BaseEntity{
			ID:   RoadAccidentIDPrefix + entityID,
			Type: RoadAccidentTypeName,
			Context: []string{
				"https://raw.githubusercontent.com/smart-data-models/dataModel.Transportation/master/context.jsonld",
			},
		},
	}

	return ra
}

const urnPrefix string = "urn:ngsi-ld:"

const RoadAccidentTypeName string = "RoadAccident"

const RoadAccidentIDPrefix string = urnPrefix + RoadAccidentTypeName + ":"

/*
this file is just temporary, all of this will be moved to ngsi-ld when appropriate
*/
