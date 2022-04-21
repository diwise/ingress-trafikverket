package trafficsvc

type tfvGeometry struct {
	WGS84 string `json:"WGS84"`
}

type tfvDeviation struct {
	Id        string      `json:"Id"`
	IconId    string      `json:"IconId"`
	Geometry  tfvGeometry `json:"Geometry"`
	StartTime string      `json:"StartTime"`
	EndTime   string      `json:"EndTime"`
	Message   string      `json:"Message"`
}

type tfvResponse struct {
	Response struct {
		Result []struct {
			Situation []struct {
				Deleted   bool           `json:"Deleted"`
				Deviation []tfvDeviation `json:"Deviation"`
			} `json:"Situation"`
			Info struct {
				LastChangeID string `json:"LASTCHANGEID"`
			} `json:"INFO"`
		} `json:"RESULT"`
	} `json:"RESPONSE"`
}
