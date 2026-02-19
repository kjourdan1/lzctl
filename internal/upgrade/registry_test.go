package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.1.0", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"2.0.0", "1.9.9", 1},
		{"0.4.0", "0.3.2", 1},
		{"0.4.1", "0.4.0", 1},
		{"1.0.0", "0.99.99", 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := compareVersions(tt.a, tt.b)
			if tt.want > 0 {
				assert.Greater(t, got, 0)
			} else if tt.want < 0 {
				assert.Less(t, got, 0)
			} else {
				assert.Equal(t, 0, got)
			}
		})
	}
}

func TestSplitVersion(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"1.2.3", []int{1, 2, 3}},
		{"v1.2.3", []int{1, 2, 3}},
		{"0.4.0", []int{0, 4, 0}},
		{"1.0.0-rc1", []int{1, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitVersion(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestModuleRefString(t *testing.T) {
	ref := ModuleRef{Namespace: "Azure", Name: "avm-res-network-vnet", Provider: "azurerm"}
	assert.Equal(t, "Azure/avm-res-network-vnet/azurerm", ref.String())
}
