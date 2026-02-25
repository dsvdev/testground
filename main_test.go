package main

import "testing"

func TestGreet(t *testing.T) {
	got := Greet("World")
	want := "Hello, World!"

	if got != want {
		t.Errorf("Greet() = %q, want %q", got, want)
	}
}
