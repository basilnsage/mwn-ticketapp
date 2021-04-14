package main

import (
	"errors"
	"strconv"
)

type fakeTicketsCollection struct {
	tickets map[string]Ticket
	id      int
}

func newFakeTicketsCollection() *fakeTicketsCollection {
	return &fakeTicketsCollection{
		make(map[string]Ticket),
		0,
	}
}

func (f *fakeTicketsCollection) create(ticket Ticket) (string, error) {
	if ticket.Title == "should error" {
		return "", errors.New("unable to create ticket")
	}
	// if the ticket sets its own ID try to honor it
	if ticket.Id != "" {
		if _, ok := f.tickets[ticket.Id]; ok {
			// this ticket has already been saved with this ID
			// updating an existing ticket should be done through a different method, so error out
			return "", errors.New("ticket already exists, use the update method instead")
		}
		f.tickets[ticket.Id] = ticket
		return ticket.Id, nil
	}
	// the ticket did not set its own ID
	// iterate f.id until we find one that hasn't been taken
	var currId string
	for {
		// first check that the f.id has not already been taken
		currId = strconv.Itoa(f.id)
		if _, ok := f.tickets[currId]; ok {
			// f.id has already been taken, increment and try again
			f.id++
		} else {
			// f.id is free! use it
			break
		}
	}
	ticket.Id = currId
	f.id++
	f.tickets[currId] = ticket
	return currId, nil
}

func (f *fakeTicketsCollection) read(id string) (*Ticket, error) {
	ticket, ok := f.tickets[id]
	if !ok {
		return nil, nil
	}
	return &ticket, nil
}

func (f *fakeTicketsCollection) update(ticketId string, ticket Ticket) (bool, error) {
	if ticket.Id != "" && ticket.Id != ticketId {
		return false, errors.New("provided ticket ID does not match filter ticket ID")
	}
	if _, ok := f.tickets[ticketId]; !ok {
		return false, nil
	}
	f.tickets[ticketId] = ticket
	return true, nil
}

func (f *fakeTicketsCollection) createWrapper(title string, price float64, version uint) Ticket {
	ticket := Ticket{
		Title:   title,
		Price:   price,
		Version: version,
	}
	tid, _ := f.create(ticket)
	ticket.Id = tid
	return ticket
}
