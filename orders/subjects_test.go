package main

import "testing"

func TestSetOrderCreated(t *testing.T) {
	var createdSubj string
	if err := setOrderCreated(&createdSubj); err != nil {
		t.Fatalf("setOrderCreated erred: %v", err)
	}
	if got, want := createdSubj, "order:created"; got != want {
		t.Fatalf("wrong subject: %v, want %v", got, want)
	}
}

func TestSetOrderCancelled(t *testing.T) {
	var cancelledSubj string
	if err := setOrderCancelled(&cancelledSubj); err != nil {
		t.Fatalf("setOrderCancelled erred: %v", err)
	}
	if got, want := cancelledSubj, "order:cancelled"; got != want {
		t.Fatalf("wrong subject: %v, want %v", got, want)
	}
}
