package weathersvc

type geometry struct {
	Position string `json:"WGS84"`
}

type measurement struct {
	Air         air    `json:"Air"`
	MeasureTime string `json:"MeasureTime"`
}

type air struct {
	Temp float64 `json:"Temp"`
}

type weatherStation struct {
	ID          string      `json:"ID"`
	Name        string      `json:"Name"`
	Geometry    geometry    `json:"Geometry"`
	Measurement measurement `json:"Measurement"`
}
type weatherStationResponse struct {
	Response struct {
		Result []struct {
			WeatherStations []weatherStation `json:"WeatherStation"`
			Info            struct {
				LastChangeID string `json:"LASTCHANGEID"`
			} `json:"INFO"`
		} `json:"RESULT"`
	} `json:"RESPONSE"`
}
