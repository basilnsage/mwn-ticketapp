package main

import (
	"github.com/basilnsage/mwn-ticketapp-common/subjects"
)

var (
	ticketCreatedSubject  string
	ticketUpdatedSubject  string
	orderCreatedSubject   string
	orderCancelledSubject string
)

func setTicketCreated(subj *string) error {
	tcs, err := subjects.StringifySubject(subjects.Subject_TICKET_CREATED)
	if err != nil {
		return err
	}
	*subj = tcs
	return nil
}

func setTicketUpdated(subj *string) error {
	tus, err := subjects.StringifySubject(subjects.Subject_TICKET_UPDATED)
	if err != nil {
		return err
	}
	*subj = tus
	return nil
}

func setTicketSubjects() error {
	if err := setTicketCreated(&ticketCreatedSubject); err != nil {
		return err
	}
	if err := setTicketUpdated(&ticketUpdatedSubject); err != nil {
		return err
	}
	return nil
}

func setOrderCreated(subj *string) error {
	ocs, err := subjects.StringifySubject(subjects.Subject_ORDER_CREATED)
	if err != nil {
		return err
	}
	*subj = ocs
	return nil
}

func setOrderCancelled(subj *string) error {
	ocs, err := subjects.StringifySubject(subjects.Subject_ORDER_CANCELLED)
	if err != nil {
		return err
	}
	*subj = ocs
	return nil
}

func setOrderSubjects() error {
	if err := setOrderCreated(&orderCreatedSubject); err != nil {
		return err
	}
	if err := setOrderCancelled(&orderCancelledSubject); err != nil {
		return err
	}
	return nil
}
