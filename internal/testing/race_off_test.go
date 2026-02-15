//go:build !race
// +build !race

package testing

import gotesting "testing"

func TestIsRaceOn_Off(got *gotesting.T) {
	if IsRaceOn() {
		got.Fatalf("expected race detector to be off")
	}
}
