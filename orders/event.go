package main

import (
	"github.com/basilnsage/mwn-ticketapp-common/events"
	"github.com/basilnsage/mwn-ticketapp-common/subjects"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/proto"
)

func marshalOrderCreated(ticket Ticket, order Order) ([]byte, error) {
	// convert expiresAt into a proto-compatible format
	pbExpiresAt, err := ptypes.TimestampProto(order.ExpiresAt)
	if err != nil {
		return nil, err
	}

	// define the event
	createdEvent := &events.OrderCreated{
		Subject: subjects.Subject_ORDER_CREATED,
		Data: &events.CreatedData{
			Id:        order.Id,
			Status:    events.Status_Created,
			UserId:    order.UserId,
			ExpiresAt: pbExpiresAt,
			Ticket: &events.CreatedData_Ticket{
				Id:    ticket.Id,
				Price: ticket.Price,
			},
		},
	}

	// marshal the event proto
	createdEventBytes, err := proto.Marshal(createdEvent)
	if err != nil {
		return nil, err
	}

	return createdEventBytes, nil
}

func marshalOrderCancelled(ticket Ticket, order Order) ([]byte, error) {
	cancelledEvent := &events.OrderCancelled{
		Subject: subjects.Subject_ORDER_CANCELLED,
		Data: &events.CancelledData{
			Id: order.Id,
			Ticket: &events.CancelledData_Ticket{
				Id: ticket.Id,
			},
		},
	}

	cancelledEventBytes, err := proto.Marshal(cancelledEvent)
	if err != nil {
		return nil, err
	}
	return cancelledEventBytes, nil
}
