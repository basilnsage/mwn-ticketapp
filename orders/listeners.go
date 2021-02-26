package main

import (
	"github.com/nats-io/stan.go"
)

type natsListener struct {
	tc       ticketsCRUD
	natsConn stan.Conn
	subs     []stan.Subscription
}

func newNatsListener(ticketsCRUD ticketsCRUD, natsConn stan.Conn) natsListener {
	var subs []stan.Subscription
	return natsListener{ticketsCRUD, natsConn, subs}
}

func (nl natsListener) close() error {
	for _, sub := range nl.subs {
		_ = sub.Close()
	}
	return nil
}

func (nl natsListener) handleTicketCreated(msg *stan.Msg) {
	_ = nl.tc
	// do something...
}

func (nl natsListener) listenForEvents() error {
	ticketCreatedSub, err := nl.natsConn.Subscribe("asdf", nl.handleTicketCreated)
	if err != nil {
		return err
	}
	nl.subs = append(nl.subs, ticketCreatedSub)

	return nil
}
