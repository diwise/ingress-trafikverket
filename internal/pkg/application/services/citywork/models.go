package citywork

type sdlGeometry struct {
	Type       string `json:"type"`
	Geometries []struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometries"`
}

type sdlFeature struct {
	Type       string      `json:"type"`
	Geometry   sdlGeometry `json:"geometry"`
	Properties struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		Restrictions string `json:"restrictions"`
		Level        string `json:"level"`
		Start        string `json:"start"`
		End          string `json:"end"`
	} `json:"properties"`
}

type sdlResponse struct {
	Type string `json:"type"`
	Name string `json:"name"`
	CRS  struct {
		Type       string `json:"type"`
		Properties struct {
			Name string `json:"name"`
		} `json:"properties"`
	} `json:"crs"`
	Features []sdlFeature `json:"features"`
}
