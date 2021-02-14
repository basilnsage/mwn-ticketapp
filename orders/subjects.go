package main

import (
	"github.com/basilnsage/mwn-ticketapp-common/subjects"
)

var (
	orderCreatedSubject   string
	orderCancelledSubject string
)

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
