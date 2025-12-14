package validator

import (
	"testing"
)

type validationTestCase struct {
	name    string
	value   string
	wantErr bool
}

func runValidationTests(t *testing.T, tag string, tests []validationTestCase) {
	v := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.value, tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s validation for '%s': error = %v, wantErr %v", tag, tt.value, err, tt.wantErr)
			}
		})
	}
}
