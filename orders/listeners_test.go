package main

import (
	"flag"
	"github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/stan.go/pb"
	"testing"

	"github.com/basilnsage/mwn-ticketapp-common/events"
	"github.com/google/go-cmp/cmp"
	"github.com/nats-io/stan.go"
	"google.golang.org/protobuf/proto"
)

func newTestNatsServer() (*server.StanServer, error) {
	fs := flag.NewFlagSet("nats-streaming-server-flags", flag.ContinueOnError)
	_ = fs.String("-p", "8765", "")
	_ = fs.String("-m", "8764", "")
	_ = fs.String("-hbi", "5s", "")
	_ = fs.String("-hbt", "5s", "")
	_ = fs.String("-hbf", "2", "")
	_ = fs.String("-SD", "", "")
	_ = fs.String("-cid", "test-ticketing-server", "")

	if err := fs.Parse([]string{}); err != nil {
		return nil, err
	}

	stanOpts, natsOpts, err := server.ConfigureOptions(fs, []string{}, func() {}, func() {}, func() {})
	if err != nil {
		return nil, err
	}

	stanServer, err := server.Run(stanOpts, natsOpts)
	if err != nil {
		return nil, err
	}

	return stanServer, nil
}

func newTestNatsListener() (func(), natsListener, error) {
	stanServer, err := newTestNatsServer()
	if err != nil {
		return nil, natsListener{}, err
	}

	natsConn, err := stan.Connect(stanServer.ClusterID(), "clientId", stan.NatsURL(stanServer.ClientURL()))
	if err != nil {
		return nil, natsListener{}, err
	}

	fakeTicketsCollection := newFakeTicketsCollection()
	testNatsListener := newNatsListener(fakeTicketsCollection, natsConn)
	return stanServer.Shutdown, testNatsListener, nil
}

func TestListenForEvents(t *testing.T) {
	if err := setTicketSubjects(); err != nil {
		t.Fatalf("unable to set ticket event subjects: %v", err)
	}

	shutdown, testNatsListener, err := newTestNatsListener()
	if err != nil {
		t.Fatalf("cound not init test nats infra: %v", err)
	}
	defer shutdown()

	if err := testNatsListener.listenForEvents(); err != nil {
		t.Fatalf("natsListener.listenForEvents: %v", err)
	}
	if err := testNatsListener.close(); err != nil {
		t.Fatalf("natsListener.close: %v", err)
	}
}

func TestHandleTicketCreated(t *testing.T) {
	testNatsListener := newNatsListener(newFakeTicketsCollection(), nil)

	fakeTicketProto := &events.CreateUpdateTicket{
		Title: "coffee ticket",
		Price: 10.0,
		Id:    "1",
		Owner: "1",
	}
	fakeTicketBytes, err := proto.Marshal(fakeTicketProto)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	msg := &stan.Msg{
		MsgProto: pb.MsgProto{
			Data: fakeTicketBytes,
		},
	}
	if err := testNatsListener.handleTicketCreated(msg); err != nil {
		t.Fatalf("error handling ticket created event: %v", err)
	}

	got, _ := testNatsListener.tc.read("1")
	want := Ticket{
		"coffee ticket",
		10.0,
		0,
		"1",
	}
	if diff := cmp.Diff(want, *got); diff != "" {
		t.Fatalf("diff: (-want +got)\n%v", diff)
	}
}

func TestHandleTicketUpdated(t *testing.T) {
	testNatsListener := newNatsListener(newFakeTicketsCollection(), nil)

	ticket := Ticket{
		"update me",
		100.5,
		0,
		"1",
	}
	_, _ = testNatsListener.tc.create(ticket)
	ticket.Title = "am i updated now"
	ticket.Price = 9.99
	fakeTicketProto := &events.CreateUpdateTicket{
		Title: ticket.Title,
		Price: ticket.Price,
		Id:    ticket.Id,
		Owner: "1",
	}
	fakeTicketBytes, err := proto.Marshal(fakeTicketProto)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	msg := &stan.Msg{
		MsgProto: pb.MsgProto{
			Data: fakeTicketBytes,
		},
	}
	if err := testNatsListener.handleTicketUpdated(msg); err != nil {
		t.Fatalf("handleTicketUpdated: %v", err)
	}

	got, _ := testNatsListener.tc.read(ticket.Id)
	if diff := cmp.Diff(ticket, *got); diff != "" {
		t.Fatalf("diff: (-want +got)\n%v", diff)
	}
}
