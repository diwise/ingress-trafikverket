package trafficsvc

type tfvGeometry struct {
	WGS84 string `json:"WGS84"`
}

type tfvDeviation struct {
	Id       string      `json:"Id"`
	Header   string      `json:"Header"`
	IconId   string      `json:"IconId"`
	Geometry tfvGeometry `json:"Geometry"`
}

type tfvResponse struct {
	Response struct {
		Result []struct {
			Situation []struct {
				Deviation []tfvDeviation `json:"Deviation"`
			} `json:"Situation"`
		} `json:"RESULT"`
	} `json:"RESPONSE"`
}
