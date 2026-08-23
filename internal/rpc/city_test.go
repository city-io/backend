package rpc

import (
	"testing"

	"cityio/internal/constants"
)

func TestValidateCityPolicy(t *testing.T) {
	tests := []struct {
		name           string
		militiaTarget  float64
		taxRatePercent int
		wantErr        bool
	}{
		{name: "exact militia target", militiaTarget: 302, taxRatePercent: 0},
		{name: "negative militia target", militiaTarget: -1, taxRatePercent: 10, wantErr: true},
		{name: "fractional militia target", militiaTarget: 25.5, taxRatePercent: 10, wantErr: true},
		{name: "negative tax", militiaTarget: 25, taxRatePercent: -1, wantErr: true},
		{name: "tax above maximum", militiaTarget: 25, taxRatePercent: constants.MaxTaxRatePercent + 1, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCityPolicy(test.militiaTarget, test.taxRatePercent)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateCityPolicy(%v, %d) error = %v, wantErr %v", test.militiaTarget, test.taxRatePercent, err, test.wantErr)
			}
		})
	}
}
