package weathersvc

type geometry struct {
	Position string `json:"WGS84"`
}

type measurement struct {
	Air         air    `json:"Air"`
	Wind        wind   `json:"Wind"`
	MeasureTime string `json:"MeasureTime"`
}

type air struct {
	Temp             float64 `json:"Temp"`
	RelativeHumidity float64 `json:"RelativeHumidity"`
}

type wind struct {
	Direction int     `json:"Direction"`
	Force     float64 `json:"Force"`
	ForceMax  float64 `json:"ForceMax"`
}

type weatherStation struct {
	ID          string      `json:"ID"`
	Name        string      `json:"Name"`
	Active      bool        `json:"Active"`
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
