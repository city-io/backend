package rpc

import (
	"testing"

	"cityio/internal/constants"
)

func TestValidateCityPolicy(t *testing.T) {
	tests := []struct {
		name            string
		garrisonPercent int
		taxRatePercent  int
		wantErr         bool
	}{
		{name: "minimum garrison", garrisonPercent: constants.MinGarrisonPercent, taxRatePercent: 0},
		{name: "maximum policy", garrisonPercent: constants.MaxGarrisonPercent, taxRatePercent: constants.MaxTaxRatePercent},
		{name: "garrison below minimum", garrisonPercent: constants.MinGarrisonPercent - 1, taxRatePercent: 10, wantErr: true},
		{name: "garrison above maximum", garrisonPercent: constants.MaxGarrisonPercent + 1, taxRatePercent: 10, wantErr: true},
		{name: "negative tax", garrisonPercent: constants.DefaultGarrisonPercent, taxRatePercent: -1, wantErr: true},
		{name: "tax above maximum", garrisonPercent: constants.DefaultGarrisonPercent, taxRatePercent: constants.MaxTaxRatePercent + 1, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCityPolicy(test.garrisonPercent, test.taxRatePercent)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateCityPolicy(%d, %d) error = %v, wantErr %v", test.garrisonPercent, test.taxRatePercent, err, test.wantErr)
			}
		})
	}
}
