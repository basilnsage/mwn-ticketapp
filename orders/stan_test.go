package main

import (
	"errors"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
)

type fakeNatsConn struct {
	messages map[string][][]byte
}

func newFakeNatsConn() *fakeNatsConn {
	return &fakeNatsConn{
		make(map[string][][]byte),
	}
}

func (f *fakeNatsConn) Publish(subj string, data []byte) error {
	f.messages[subj] = append(f.messages[subj], data)
	return nil
}

func (f *fakeNatsConn) PublishAsync(subj string, data []byte, ah stan.AckHandler) (string, error) {
	_, _, _ = subj, data, ah
	return "", errors.New("not implemented")
}

func (f *fakeNatsConn) Subscribe(subj string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	_, _, _ = subj, cb, opts
	return nil, errors.New("not implemented")
}

func (f *fakeNatsConn) QueueSubscribe(subj, qgroup string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (stan.Subscription, error) {
	_, _, _, _ = subj, qgroup, cb, opts
	return nil, errors.New("not implemented")
}

func (f *fakeNatsConn) Close() error {
	return nil
}

func (f *fakeNatsConn) NatsConn() *nats.Conn {
	return nil
}
