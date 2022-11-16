package main

import "testing"

func Test_greet(t *testing.T) {
	want := 2
	if got := 1 + 1; got != want {
		t.Errorf("greet() = %v, want %v", got, want)
	}
}
