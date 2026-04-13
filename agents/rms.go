package agents

import (
	"encoding/binary"
	"math"
)

// RMSFromPCM computes a normalised RMS amplitude in [0.0, 1.0] from a
// 16-bit little-endian mono PCM buffer — the format malgo delivers in the
// onSamples duplex callback.
func RMSFromPCM(pcm []byte) float64 {
	n := len(pcm) / 2
	if n == 0 {
		return 0
	}

	var sumSq float64
	for i := 0; i < n; i++ {
		sample := int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
		f := float64(sample) / 32768.0
		sumSq += f * f
	}

	rms := math.Sqrt(sumSq / float64(n))

	// Scale up so a normal speaking voice sits comfortably in the green zone.
	// Clamp to [0, 1].
	return math.Min(rms*5.0, 1.0)
}
