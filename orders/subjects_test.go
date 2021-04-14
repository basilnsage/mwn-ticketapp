package main

import "testing"

func TestSetTicketCreated(t *testing.T) {
	var createdSubj string
	if err := setTicketCreated(&createdSubj); err != nil {
		t.Fatalf("setTicketCreated: %v", err)
	}
	if got, want := createdSubj, "ticket:created"; got != want {
		t.Fatalf("wrong subject: %v, want %v", got, want)
	}
}

func TestSetTicketUpdated(t *testing.T) {
	var updatedSubj string
	if err := setTicketUpdated(&updatedSubj); err != nil {
		t.Fatalf("setTicketCreated: %v", err)
	}
	if got, want := updatedSubj, "ticket:updated"; got != want {
		t.Fatalf("wrong subject: %v, want %v", got, want)
	}
}

func TestSetOrderCreated(t *testing.T) {
	var createdSubj string
	if err := setOrderCreated(&createdSubj); err != nil {
		t.Fatalf("setOrderCreated: %v", err)
	}
	if got, want := createdSubj, "order:created"; got != want {
		t.Fatalf("wrong subject: %v, want %v", got, want)
	}
}

func TestSetOrderCancelled(t *testing.T) {
	var cancelledSubj string
	if err := setOrderCancelled(&cancelledSubj); err != nil {
		t.Fatalf("setOrderCancelled: %v", err)
	}
	if got, want := cancelledSubj, "order:cancelled"; got != want {
		t.Fatalf("wrong subject: %v, want %v", got, want)
	}
}
