package fiware

import (
	"github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/geojson"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
)

type CityWork struct {
	ngsitypes.BaseEntity
	Location     *geojson.GeoJSONProperty    `json:"location,omitempty"`
	Description  *ngsitypes.TextProperty     `json:"description,omitempty"`
	DateCreated  ngsitypes.DateTimeProperty  `json:"dateCreated"`
	DateModified *ngsitypes.DateTimeProperty `json:"dateModified,omitempty"`
	EndDate      ngsitypes.DateTimeProperty  `json:"endDate"`
	StartDate    ngsitypes.DateTimeProperty  `json:"startDate"`
}

func NewCityWork(entityID string) CityWork {
	cw := CityWork{
		BaseEntity: ngsitypes.BaseEntity{
			ID:   CityWorkIDPrefix + entityID,
			Type: CityWorkTypeName,
			Context: []string{
				"https://raw.githubusercontent.com/smart-data-models/dataModel.Transportation/master/context.jsonld",
			},
		},
	}

	return cw
}

const CityWorkTypeName string = "CityWork"
const CityWorkIDPrefix string = urnPrefix + CityWorkTypeName + ":"
