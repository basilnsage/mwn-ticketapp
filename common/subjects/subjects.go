package subjects

import "errors"

var protoSubjToString = map[string]string{
	"TicketCreated": "ticket:created",
	"TicketUpdated": "ticket:updated",
	"OrderCreated": "order:created",
	"OrderCancelled": "order:cancelled",
}

var stringToProtoSubj = map[string]string {
	"ticket:created": "TicketCreated",
	"ticket:updated": "TicketUpdated",
	"order:created": "OrderCreated",
	"order:cancelled": "OrderCancelled",
}

func StringifySubject(enum int32) (string, error) {
	protoSubj, ok := Subject_name[enum]
	if !ok {
		return "", errors.New("invalid subject")
	}
	subject, _ := protoSubjToString[protoSubj]
	return subject, nil
}

func SubjectifyString(subject string) (int32, error) {
	protoSubj, ok := stringToProtoSubj[subject]
	if !ok {
		return -1, errors.New("invalid subject")
	}
	enum, _ := Subject_value[protoSubj]
	return enum, nil
}