package lttb

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestLTTB(t *testing.T) {
	input := []Point{
		{0.0, 0.8784277338451055},
		{1.0, 0.1499321530254274},
		{2.0, 0.49489164039056865},
		{3.0, 0.296325207554909},
		{4.0, 0.7954332017957191},
		{5.0, 0.37920694146084544},
		{6.0, 0.262280416971284},
		{7.0, 0.2115412028253334},
		{8.0, 0.6010906649928144},
		{9.0, 0.8143458607144891},
	}

	expected := []Point{
		input[0],
		input[1],
		input[4],
		input[7],
		input[9],
	}

	actual := Downsample(input, 5)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unexpected result from LTTB, expected %v, actual %v", expected, actual)
	}
}
