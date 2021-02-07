package events

const (
	createTicketSubj = "ticket:created"
	updateTicketSubj = "ticket:updated"
)

type Subject struct{}

func (s Subject) CreateTicket() string {
	return createTicketSubj
}
func (s Subject) UpdateTicket() string {
	return updateTicketSubj
}
