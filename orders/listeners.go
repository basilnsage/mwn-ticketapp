package main

import (
	"fmt"

	"github.com/basilnsage/mwn-ticketapp-common/events"
	"github.com/nats-io/stan.go"
	"google.golang.org/protobuf/proto"
)

var queueGroupName = "orders-svc"

type natsListener struct {
	tc       ticketsCRUD
	stanConn stan.Conn
	subs     []stan.Subscription
}

func newNatsListener(ticketsCRUD ticketsCRUD, stanConn stan.Conn) natsListener {
	var subs []stan.Subscription
	return natsListener{ticketsCRUD, stanConn, subs}
}

func (nl natsListener) close() error {
	for _, sub := range nl.subs {
		if err := sub.Close(); err != nil {
			ErrorLogger.Printf("error closing NATS subscription: %v", err)
		}
	}
	return nil
}

func (nl natsListener) listenForEvents() error {
	if err := setTicketSubjects(); err != nil {
		return fmt.Errorf("could not set ticket event subjects: %v", err)
	}

	ticketCreatedSub, err := nl.stanConn.QueueSubscribe(ticketCreatedSubject, queueGroupName, func(msg *stan.Msg) {
		if err := nl.handleTicketCreated(msg); err != nil {
			ErrorLogger.Printf("%v: error handling event: %v", ticketCreatedSubject, err)
		} else {
			if err := msg.Ack(); err != nil {
				ErrorLogger.Printf("%v: error acking message: %v", ticketCreatedSubject, err)
			}
		}
	}, stan.SetManualAckMode())
	if err != nil {
		return err
	}
	nl.subs = append(nl.subs, ticketCreatedSub)
	InfoLogger.Printf("%v listener subscribed", ticketCreatedSubject)

	ticketUpdatedSub, err := nl.stanConn.QueueSubscribe(ticketUpdatedSubject, queueGroupName, func(msg *stan.Msg) {
		if err := nl.handleTicketUpdated(msg); err != nil {
			ErrorLogger.Printf("%v: error handling event: %v", ticketUpdatedSubject, err)
		} else {
			if err := msg.Ack(); err != nil {
				ErrorLogger.Printf("%v: error acking message: %v", ticketUpdatedSubject, err)
			}
		}
	}, stan.SetManualAckMode())
	if err != nil {
		return err
	}
	nl.subs = append(nl.subs, ticketUpdatedSub)
	InfoLogger.Printf("%v listener subscribed", ticketUpdatedSubject)

	return nil
}

func (nl natsListener) handleTicketCreated(msg *stan.Msg) error {
	InfoLogger.Printf("%v handler received an event", ticketCreatedSubject)

	// unmarshal data into ticket created event proto
	ticketEvent := &events.CreateUpdateTicket{}
	if err := proto.Unmarshal(msg.Data, ticketEvent); err != nil {
		return fmt.Errorf("proto.Unmarshal: %v", err)
	}

	// construct ticket from proto contents
	ticket := Ticket{
		ticketEvent.Title,
		ticketEvent.Price,
		uint(0),
		ticketEvent.Id,
	}

	// save ticket to DB
	if _, err := nl.tc.create(ticket); err != nil {
		return fmt.Errorf("unable to save ticket to DB: %v", err)
	}

	InfoLogger.Printf("saved ticket created event for ticket id: %v", ticketEvent.Id)
	return nil
}

func (nl natsListener) handleTicketUpdated(msg *stan.Msg) error {
	InfoLogger.Printf("%v handler received an event", ticketUpdatedSubject)

	// unmarshal data into ticket updated event proto
	ticketEvent := &events.CreateUpdateTicket{}
	if err := proto.Unmarshal(msg.Data, ticketEvent); err != nil {
		return fmt.Errorf("proto.Unmarshal: %v", err)
	}

	// construct ticket from proto contents
	ticket := Ticket{
		ticketEvent.Title,
		ticketEvent.Price,
		0,
		ticketEvent.Id,
	}

	// save ticket to DB
	if ok, err := nl.tc.update(ticket.Id, ticket); err != nil {
		return fmt.Errorf("unable to save ticket to DB: %v", err)
	} else if !ok {
		return fmt.Errorf("tried to update a ticket that does not exist, id: %v", ticket.Id)
	}

	InfoLogger.Printf("saved ticket updated event for ticket id: %v", ticketEvent.Id)
	return nil
}
