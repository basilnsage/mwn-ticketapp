package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

var allBalls = time.Unix(0, 0)

type fakeOrdersCollection struct {
	orders map[string]Order
	id     int
}

func newFakeOrdersCollection() *fakeOrdersCollection {
	return &fakeOrdersCollection{
		make(map[string]Order),
		0,
	}
}

func (f *fakeOrdersCollection) create(order Order) (string, error) {
	currId := strconv.Itoa(f.id)
	order.Id = currId
	order.ExpiresAt = allBalls
	f.id++
	f.orders[currId] = order
	return currId, nil
}

func (f *fakeOrdersCollection) read(id string) (*Order, error) {
	order, ok := f.orders[id]
	if !ok {
		return nil, fmt.Errorf("order does not exist, id: %v", id)
	}
	return &order, nil
}

func (f *fakeOrdersCollection) searchTest(limit int64, ticketIds, userIds []string, statuses []orderStatus) ([]Order, error) {
	ticketMap := make(map[string]struct{})
	for _, tid := range ticketIds {
		ticketMap[tid] = struct{}{}
	}

	userMap := make(map[string]struct{})
	for _, uid := range userIds {
		userMap[uid] = struct{}{}
	}

	statusMap := make(map[string]struct{})
	for _, s := range statuses {
		statusMap[s.String()] = struct{}{}
	}

	var res []Order
	for _, order := range f.orders {
		if limit > 0 && len(res) > int(limit) {
			break
		}
		ticketIdOk, userIdOk, statusOk := true, true, true
		if len(ticketIds) > 0 {
			_, ticketIdOk = ticketMap[order.TicketId]
		}
		if len(userIds) > 0 {
			_, userIdOk = userMap[order.UserId]
		}
		if len(statuses) > 0 {
			_, statusOk = statusMap[order.Status.String()]
		}
		if ticketIdOk && userIdOk && statusOk {
			res = append(res, order)
		}
	}
	return res, nil
}

func (f *fakeOrdersCollection) search(limit uint, ticketId string, status ...orderStatus) ([]Order, error) {
	statuses := make(map[string]struct{})
	for _, s := range status {
		statuses[s.String()] = struct{}{}
	}

	var res []Order
	for _, value := range f.orders {
		if uint(len(res)) > limit {
			break
		}
		if value.TicketId == ticketId {
			if _, ok := statuses[value.Status.String()]; ok {
				res = append(res, value)
			}
		}
	}
	return res, nil
}

func (f *fakeOrdersCollection) update(id string, order Order) (bool, error) {
	if order.Id != "" && order.Id != id {
		return false, errors.New("provided order does not match provided order ID")
	}
	if _, ok := f.orders[id]; !ok {
		return false, fmt.Errorf("order does not exist, id: %v", id)
	}
	f.orders[id] = order
	return true, nil
}

// a wrapper around the create method
func (f *fakeOrdersCollection) createWrapper(uid, tid string, status orderStatus) Order {
	order := Order{
		UserId: uid,
		Status: status,
		ExpiresAt: allBalls,
		TicketId: tid,
	}
	oid, _  := f.create(order)
	order.Id = oid
	return order
}