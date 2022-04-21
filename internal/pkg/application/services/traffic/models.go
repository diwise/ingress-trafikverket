package trafficsvc

type tfvGeometry struct {
	WGS84 string `json:"WGS84"`
}

type tfvDeviation struct {
	Id        string      `json:"Id"`
	Header    string      `json:"Header"`
	IconId    string      `json:"IconId"`
	Geometry  tfvGeometry `json:"Geometry"`
	StartTime string      `json:"StartTime"`
	EndTime   string      `json:"EndTime"`
}

type tfvResponse struct {
	Response struct {
		Result []struct {
			Situation []struct {
				Deviation []tfvDeviation `json:"Deviation"`
			} `json:"Situation"`
			Info struct {
				LastChangeID string `json:"LASTCHANGEID"`
			} `json:"INFO"`
		} `json:"RESULT"`
	} `json:"RESPONSE"`
}
