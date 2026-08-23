package rpc

import (
	"testing"

	"cityio/internal/constants"
)

func TestValidateCityPolicy(t *testing.T) {
	tests := []struct {
		name           string
		militiaPercent int
		taxRatePercent int
		wantErr        bool
	}{
		{name: "minimum militia", militiaPercent: constants.MinMilitiaPercent, taxRatePercent: 0},
		{name: "maximum policy", militiaPercent: constants.MaxMilitiaPercent, taxRatePercent: constants.MaxTaxRatePercent},
		{name: "militia below minimum", militiaPercent: constants.MinMilitiaPercent - 1, taxRatePercent: 10, wantErr: true},
		{name: "militia above maximum", militiaPercent: constants.MaxMilitiaPercent + 1, taxRatePercent: 10, wantErr: true},
		{name: "negative tax", militiaPercent: constants.DefaultMilitiaPercent, taxRatePercent: -1, wantErr: true},
		{name: "tax above maximum", militiaPercent: constants.DefaultMilitiaPercent, taxRatePercent: constants.MaxTaxRatePercent + 1, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCityPolicy(test.militiaPercent, test.taxRatePercent)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateCityPolicy(%d, %d) error = %v, wantErr %v", test.militiaPercent, test.taxRatePercent, err, test.wantErr)
			}
		})
	}
}
