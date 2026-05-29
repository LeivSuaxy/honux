package geometry

const (
	Rectangle = "rectangle"
	Polygon   = "polygon"
	Circle    = "circle"
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PolygonGeometry struct {
	Points []Point `json:"points"`
}

type RectangleGeometry struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type CircleGeometry struct {
	Center Point   `json:"center"`
	Radius float64 `json:"radius"`
}
