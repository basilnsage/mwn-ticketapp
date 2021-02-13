// a thin wrapper around the .../common/subjects for convenience and testability
package main

import "github.com/basilnsage/mwn-ticketapp-common/subjects"

var (
	createTicketSubject string
	updateTicketSubject string
)

func setCreateTicketSubject(receiver *string) error {
	cts, err := subjects.StringifySubject(subjects.Subject_TICKET_CREATED)
	if err != nil {
		return err
	}
	*receiver = cts
	return nil
}

func setUpdateTicketSubject(receiver *string) error {
	uts, err := subjects.StringifySubject(subjects.Subject_TICKET_UPDATED)
	if err != nil {
		return err
	}
	*receiver = uts
	return nil
}

func setSubjects() error {
	if err := setCreateTicketSubject(&createTicketSubject); err != nil {
		return err
	}
	if err := setUpdateTicketSubject(&updateTicketSubject); err != nil {
		return err
	}
	return nil
}
