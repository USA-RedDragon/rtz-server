package utils_test

import (
	"fmt"
	"testing"

	"github.com/USA-RedDragon/rtz-server/internal/utils"
)

func TestLuhn(t *testing.T) {
	t.Parallel()

	validLuhns := []int{
		0,
		7082049359,
		5753425163,
		711087701572692,
	}

	invalidLuhns := []int{
		1,
		7082049358,
		5753425164,
		711087701572693,
	}

	for _, tt := range validLuhns {
		tt := tt
		t.Run(fmt.Sprintf("LuhnValid[%d]", tt), func(t *testing.T) {
			t.Parallel()
			if got := utils.LuhnValid(tt); !got {
				t.Errorf("LuhnValid() = %v, want %v", got, true)
			}
		})
	}

	for _, tt := range invalidLuhns {
		tt := tt
		t.Run(fmt.Sprintf("LuhnInValid[%d]", tt), func(t *testing.T) {
			t.Parallel()
			if got := utils.LuhnValid(tt); got {
				t.Errorf("LuhnInvalidValid() = %v, want %v", got, false)
			}
		})
	}
}
