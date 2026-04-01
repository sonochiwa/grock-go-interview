package shape_calculator

import "math"

type Shape interface {
	Area() float64
	Perimeter() float64
}

type Circle struct {
	Radius float64
}

// TODO: реализуй Area и Perimeter для Circle
// Area = π * r²
// Perimeter = 2 * π * r

type Rectangle struct {
	Width  float64
	Height float64
}

// TODO: реализуй Area и Perimeter для Rectangle

// TODO: реализуй TotalArea
func TotalArea(shapes ...Shape) float64 {
	_ = math.Pi
	return 0
}
