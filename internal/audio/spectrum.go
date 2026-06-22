package audio

import (
	"encoding/binary"
	"math"
)

type SpectrumAnalyzer struct {
	sampleRate int
	frequency  []float64
}

func NewSpectrumAnalyzer(sampleRate, bands int, low, high float64) SpectrumAnalyzer {
	frequencies := make([]float64, bands)
	ratio := math.Pow(high/low, 1/float64(bands-1))
	for i := range frequencies {
		frequencies[i] = low * math.Pow(ratio, float64(i))
	}
	return SpectrumAnalyzer{sampleRate: sampleRate, frequency: frequencies}
}

func (a SpectrumAnalyzer) Bands(buf []byte) []float64 {
	samples := pcm(buf)
	out := make([]float64, len(a.frequency))
	if len(samples) == 0 {
		return out
	}
	for i, freq := range a.frequency {
		out[i] = a.goertzel(samples, freq)
	}
	normalize(out)
	return out
}

func (a SpectrumAnalyzer) goertzel(samples []float64, freq float64) float64 {
	k := 0.5 + (float64(len(samples))*freq)/float64(a.sampleRate)
	omega := (2 * math.Pi * math.Floor(k)) / float64(len(samples))
	coef := 2 * math.Cos(omega)
	q1, q2 := 0.0, 0.0
	for i, sample := range samples {
		window := 0.5 - 0.5*math.Cos((2*math.Pi*float64(i))/float64(len(samples)-1))
		q0 := sample*window + coef*q1 - q2
		q2 = q1
		q1 = q0
	}
	power := q1*q1 + q2*q2 - q1*q2*coef
	return math.Sqrt(math.Max(0, power)) / float64(len(samples))
}

func pcm(buf []byte) []float64 {
	samples := make([]float64, 0, len(buf)/2)
	for i := 0; i+1 < len(buf); i += 2 {
		v := float64(int16(binary.LittleEndian.Uint16(buf[i:]))) / 32768
		samples = append(samples, v)
	}
	return samples
}

func normalize(values []float64) {
	maxValue := 0.0
	for i, value := range values {
		values[i] = math.Sqrt(value) * 5
		if values[i] > maxValue {
			maxValue = values[i]
		}
	}
	if maxValue < 0.001 {
		return
	}
	for i := range values {
		values[i] = math.Min(1, values[i]/maxValue)
	}
}
