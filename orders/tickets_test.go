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
	currId := strconv.Itoa(f.id)
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
