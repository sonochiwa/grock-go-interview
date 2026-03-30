package shape_calculator

import (
	"math"
	"testing"
)

const epsilon = 1e-9

func TestCircleArea(t *testing.T) {
	c := Circle{Radius: 5}
	want := math.Pi * 25
	if got := c.Area(); math.Abs(got-want) > epsilon {
		t.Errorf("Circle{5}.Area() = %f, want %f", got, want)
	}
}

func TestCirclePerimeter(t *testing.T) {
	c := Circle{Radius: 5}
	want := 2 * math.Pi * 5
	if got := c.Perimeter(); math.Abs(got-want) > epsilon {
		t.Errorf("Circle{5}.Perimeter() = %f, want %f", got, want)
	}
}

func TestRectangleArea(t *testing.T) {
	r := Rectangle{Width: 4, Height: 6}
	want := 24.0
	if got := r.Area(); math.Abs(got-want) > epsilon {
		t.Errorf("Rectangle{4,6}.Area() = %f, want %f", got, want)
	}
}

func TestRectanglePerimeter(t *testing.T) {
	r := Rectangle{Width: 4, Height: 6}
	want := 20.0
	if got := r.Perimeter(); math.Abs(got-want) > epsilon {
		t.Errorf("Rectangle{4,6}.Perimeter() = %f, want %f", got, want)
	}
}

func TestTotalArea(t *testing.T) {
	shapes := []Shape{
		Circle{Radius: 1},
		Rectangle{Width: 2, Height: 3},
	}
	want := math.Pi + 6.0
	if got := TotalArea(shapes...); math.Abs(got-want) > epsilon {
		t.Errorf("TotalArea() = %f, want %f", got, want)
	}
}

func TestTotalAreaEmpty(t *testing.T) {
	if got := TotalArea(); got != 0 {
		t.Errorf("TotalArea() empty = %f, want 0", got)
	}
}

func TestShapeInterface(t *testing.T) {
	var _ Shape = Circle{}
	var _ Shape = Rectangle{}
}
