package citywork

import "encoding/json"

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

/*
type sdlGeometry struct {
	Type       string `json:"type"`
	Geometries []struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometries"`
}
*/
type sdlGeometry struct {
	Type       string          `json:"type"`
	Geometries json.RawMessage `json:"geometries"`
}

/*
func (gjgi *sdlGeometry) UnmarshalJSON(data []byte) error {
	temp := struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}{}

	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}

	if temp.Type == "LineString" {
		coords := [][]float64{}
		err = json.Unmarshal(temp.Coordinates, &coords)
		if err != nil {
			return err
		}

		gjp := CreateGeoJSONPropertyFromLineString(coords)
		gjgi.Geometry = gjp.Value
	} else if temp.Type == "MultiPolygon" {
		coords := [][][][]float64{}
		err = json.Unmarshal(temp.Coordinates, &coords)
		if err != nil {
			return err
		}

		gjp := CreateGeoJSONPropertyFromMultiPolygon(coords)
		gjgi.Geometry = gjp.Value
	} else if temp.Type == "Point" {
		coords := [2]float64{}
		err = json.Unmarshal(temp.Coordinates, &coords)
		if err != nil {
			return err
		}

		gjp := CreateGeoJSONPropertyFromWGS84(coords[0], coords[1])
		gjgi.Geometry = gjp.Value
	} else {
		return fmt.Errorf("unable to unmarshal geometry of type %s", temp.Type)
	}

	return nil
}
*/
