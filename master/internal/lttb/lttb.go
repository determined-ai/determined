/*
Package lttb includes a Go rewrite of Sveinn Steinarsson's reference implementation (written in
JavaScript) of the Largest-Triangle-Three-Buckets algorithm as described in his thesis,
"Downsampling Time Series for Visual Representation":

Advisors: Jóhann Pétur Malmquist, Kristján Jónasson
Faculty Representative: Bjarni Júlíusson
Faculty of Industrial Engineering, Mechanical Engineering and Computer Science
School of Engineering and Natural Sciences
University of Iceland
Reykjavik, June 2013
https://skemman.is/bitstream/1946/15343/3/SS_MSthesis.pdf

As far as translation makes possible, the original algorithm, variable names, and comments are
unmodified. The accompanying tests were added after the Go rewrite. The following copyright notice
and the MIT license pertain to the original implementation:

Copyright (c) 2013 by Sveinn Steinarsson

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package lttb

import (
	"math"
)

// Point represents a Cartesian coordinate pair.
type Point struct {
	X float64
	Y float64
}

// Downsample selects the most visually significant points from a series.
func Downsample(data []Point, threshold int) (sampled []Point) {
	dataLength := len(data)
	if threshold >= dataLength || threshold == 0 {
		return data // Nothing to do
	}
	sampled = make([]Point, threshold)

	// Bucket size. Leave room for start and end data points
	every := float64(dataLength-2) / float64(threshold-2)

	var a, nextA int // Initially a is the first point in the triangle
	var maxArea, area float64

	sampled[0] = data[a] // Always add the first point
	sampledIndex := 1
	for i := 0; i < threshold-2; i++ {
		// Calculate point average for next bucket (containing c)
		var avgX, avgY float64
		avgRangeStart := int(float64(i+1)*every) + 1
		avgRangeEnd := int(float64(i+2)*every) + 1
		if avgRangeEnd >= dataLength {
			avgRangeEnd = dataLength
		}
		avgRangeLength := avgRangeEnd - avgRangeStart

		for ; avgRangeStart < avgRangeEnd; avgRangeStart++ {
			avgX += data[avgRangeStart].X
			avgY += data[avgRangeStart].Y
		}
		avgX /= float64(avgRangeLength)
		avgY /= float64(avgRangeLength)

		// Get the range for this bucket
		rangeOffs := int(float64(i+0)*every) + 1
		rangeTo := int(float64(i+1)*every) + 1

		// Point a
		pointAX := data[a].X
		pointAY := data[a].Y

		maxArea = -1.0

		var maxAreaPoint Point
		for ; rangeOffs < rangeTo; rangeOffs++ {
			// Calculate triangle area over three buckets
			area = math.Abs(
				(pointAX-avgX)*(data[rangeOffs].Y-pointAY)-
					(pointAX-data[rangeOffs].X)*(avgY-pointAY)) * 0.5
			if area > maxArea {
				maxArea = area
				maxAreaPoint = data[rangeOffs]
				nextA = rangeOffs // Next a is this b
			}
		}

		sampled[sampledIndex] = maxAreaPoint // Pick this point from the bucket
		sampledIndex++
		a = nextA // This a is the next a (chosen b)
	}

	sampled[sampledIndex] = data[dataLength-1] // Always add last

	return sampled
}
