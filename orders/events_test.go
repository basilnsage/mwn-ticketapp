package main

import (
	"testing"

	"github.com/basilnsage/mwn-ticketapp-common/events"
	"github.com/basilnsage/mwn-ticketapp-common/subjects"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestMarshalOrderCreated(t *testing.T) {
	ticket := Ticket{"am a ticket", 1.0, 1, "1"}
	order := Order{"1", Created, allBalls, "1", "1"}

	pbExpiresAt, err := ptypes.TimestampProto(allBalls)
	want := &events.OrderCreated{
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

	b, err := marshalOrderCreated(ticket, order)
	if err != nil {
		t.Fatalf("marshalOrderCreated: %v", err)
	}

	var got events.OrderCreated
	if err := proto.Unmarshal(b, &got); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	if diff := cmp.Diff(want, &got, protocmp.Transform()); diff != "" {
		t.Fatalf("diff: (-want +got)\n%v", diff)
	}
}

func TestMarshalOrderCancelled(t *testing.T) {
	ticket := Ticket{"am a ticket", 1.0, 1, "1"}
	order := Order{"1", Created, allBalls, "1", "1"}

	want := &events.OrderCancelled{
		Subject: subjects.Subject_ORDER_CANCELLED,
		Data: &events.CancelledData{
			Id: order.Id,
			Ticket: &events.CancelledData_Ticket{
				Id: ticket.Id,
			},
		},
	}

	b, err := marshalOrderCancelled(ticket, order)
	if err != nil {
		t.Fatalf("marshalOrderCancelled: %v", err)
	}

	var got events.OrderCancelled
	if err := proto.Unmarshal(b, &got); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	if diff := cmp.Diff(want, &got, protocmp.Transform()); diff != "" {
		t.Fatalf("diff: (-want +got)\n%v", diff)
	}
}
