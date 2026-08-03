package world

import (
	"math"
	"sort"
)

// hash2 is an integer avalanche hash. Written with wrapping uint32 arithmetic
// so it produces the same values as the JavaScript reference implementation the
// generator was tuned against.
func hash2(x, y, seed int) uint32 {
	h := uint32(int32(x))*0x27d4eb2d ^ uint32(int32(y))*0x165667b1 ^ uint32(int32(seed))*0x9e3779b9
	h = (h ^ (h >> 15)) * 0x85ebca6b
	h = (h ^ (h >> 13)) * 0xc2b2ae35
	return h ^ (h >> 16)
}

func hash01(x, y, seed int) float64 {
	return float64(hash2(x, y, seed)) / 4294967296.0
}

const r2 = 0.7071067811865476

// Eight unit directions. Gradient noise rather than value noise: a value
// lattice leaves axis-aligned artefacts that read as visibly rectangular
// coastlines once the field is thresholded into land and sea.
var gradients = [8][2]float64{
	{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	{r2, r2}, {-r2, r2}, {r2, -r2}, {-r2, -r2},
}

func fade(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

func dotGrad(ix, iy int, dx, dy float64, seed int) float64 {
	g := &gradients[hash2(ix, iy, seed)&7]
	return g[0]*dx + g[1]*dy
}

// perlin2 returns gradient noise remapped to roughly [0, 1].
func perlin2(x, y float64, seed int) float64 {
	xi, yi := int(math.Floor(x)), int(math.Floor(y))
	xf, yf := x-float64(xi), y-float64(yi)
	u, v := fade(xf), fade(yf)

	n00 := dotGrad(xi, yi, xf, yf, seed)
	n10 := dotGrad(xi+1, yi, xf-1, yf, seed)
	n01 := dotGrad(xi, yi+1, xf, yf-1, seed)
	n11 := dotGrad(xi+1, yi+1, xf-1, yf-1, seed)

	nx0 := n00 + (n10-n00)*u
	nx1 := n01 + (n11-n01)*u
	return clamp01((nx0+(nx1-nx0)*v)*r2 + 0.5)
}

// fbm sums octaves of perlin2 at increasing frequency.
func fbm(x, y float64, seed, octaves int, lacunarity, gain float64) float64 {
	amp, freq, sum, norm := 1.0, 1.0, 0.0, 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * perlin2(x*freq, y*freq, seed+i*1013)
		norm += amp
		amp *= gain
		freq *= lacunarity
	}
	return sum / norm
}

// ridged folds the signal about its midpoint, turning rounded blobs into
// creases so mountains form connected ranges rather than isolated lumps.
func ridged(x, y float64, seed, octaves int) float64 {
	amp, freq, sum, norm := 1.0, 1.0, 0.0, 0.0
	for i := 0; i < octaves; i++ {
		n := 1 - math.Abs(perlin2(x*freq, y*freq, seed+i*2087)*2-1)
		sum += amp * n * n
		norm += amp
		amp *= gain
		freq *= lacunarity
	}
	return sum / norm
}

// warp offsets a sample point by a low-frequency noise vector, which is what
// turns smooth blobby coastlines into ones with inlets and peninsulas.
func warp(x, y float64, seed int, amp float64) (float64, float64) {
	wx := fbm(x+5.2, y+1.3, seed+7717, 3, lacunarity, gain) - 0.5
	wy := fbm(x+9.1, y+4.7, seed+3313, 3, lacunarity, gain) - 0.5
	return x + wx*amp, y + wy*amp
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func smoothstep(t float64) float64 {
	c := clamp01(t)
	return c * c * (3 - 2*c)
}

// quantile returns the value at the given quantile of a sample set, sorting a
// copy rather than the input.
func quantile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 1
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	i := int(math.Round(q * float64(len(sorted)-1)))
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

// rankNormalize replaces each included value with its rank in [0, 1],
// flattening the distribution. Excluded entries are left at zero.
func rankNormalize(src []float64, include func(i int) bool) []float64 {
	idxs := make([]int, 0, len(src))
	for i := range src {
		if include(i) {
			idxs = append(idxs, i)
		}
	}
	sort.Slice(idxs, func(a, b int) bool { return src[idxs[a]] < src[idxs[b]] })
	out := make([]float64, len(src))
	denom := float64(len(idxs) - 1)
	if denom < 1 {
		denom = 1
	}
	for r, i := range idxs {
		out[i] = float64(r) / denom
	}
	return out
}
