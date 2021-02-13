package main

import "testing"

func TestSetSubjects(t *testing.T) {
	var createSubj, updateSubj string

	if err := setCreateTicketSubject(&createSubj); err != nil {
		t.Errorf("error setting createTicket subject: %v", err)
	}
	if got, want := createSubj, "ticket:created"; got != want {
		t.Errorf("incorrect createTicket subject: %v, want %v", got, want)
	}

	if err := setUpdateTicketSubject(&updateSubj); err != nil {
		t.Errorf("error setting updateTicket subject: %v", err)
	}
	if got, want := updateSubj, "ticket:updated"; got != want {
		t.Errorf("incorrect updateTicket subject: %v, want %v", got, want)
	}
}
