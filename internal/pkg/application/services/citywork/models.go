package citywork

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

type sdlGeometry struct {
	Type       string          `json:"type"`
	Geometries json.RawMessage `json:"geometries"`
}

func (sf *sdlFeature) ID() string {
	id := strings.ReplaceAll(sf.Properties.Title, " ", "") + ":" + strings.ReplaceAll(sf.Properties.Start, "-", "") + ":" + strings.ReplaceAll(sf.Properties.End, "-", "")
	return id
}

func (g *sdlGeometry) AsPoint() (float64, float64, error) {
	temp := []struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}{}

	err := json.Unmarshal(g.Geometries, &temp)
	if err != nil {
		return 0, 0, err
	}

	for _, c := range temp {
		if c.Type == "Point" {
			var p []float64
			err = json.Unmarshal(c.Coordinates, &p)
			if err != nil {
				return 0, 0, err
			}

			return p[0], p[1], nil
		}
	}

	return 0, 0, fmt.Errorf("unable to parse point")
}
