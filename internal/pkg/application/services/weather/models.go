package weathersvc

import "time"

type osv struct {
	Origin      string  `json:"Origin"`
	SensorNames string  `json:"SensorNames"`
	Value       float64 `json:"Value"`
}

type geometry struct {
	Position string `json:"WGS84"`
}

type observation struct {
	Air  *air   `json:"Air,omitempty"`
	Wind []wind `json:"Wind"`
}

type air struct {
	Temperature      osv `json:"Temperature"`
	RelativeHumidity osv `json:"RelativeHumidity"`
}

type wind struct {
	Direction *osv `json:"Direction,omitempty"`
	Speed     *osv `json:"Speed,omitempty"`
}

type weatherMeasurepoint struct {
	ID           string      `json:"Id"`
	Name         string      `json:"Name"`
	Deleted      bool        `json:"Deleted"`
	Geometry     geometry    `json:"Geometry"`
	Observation  observation `json:"Observation"`
	ModifiedTime time.Time   `json:"ModifiedTime"`
}

type weatherMeasurepointResponse struct {
	Response struct {
		Result []struct {
			WeatherMeasurepoints []weatherMeasurepoint `json:"WeatherMeasurepoint"`
			Info                 struct {
				LastChangeID string `json:"LASTCHANGEID"`
			} `json:"INFO"`
		} `json:"RESULT"`
	} `json:"RESPONSE"`
}
