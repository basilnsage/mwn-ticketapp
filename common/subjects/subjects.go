package subjects

import (
	"fmt"
)

var protoSubjToString = map[string]string{
	"TICKET_CREATED":  "ticket:created",
	"TICKET_UPDATED":  "ticket:updated",
	"ORDER_CREATED":   "order:created",
	"ORDER_CANCELLED": "order:cancelled",
}

var stringToProtoSubj = map[string]string{
	"ticket:created":  "TICKET_CREATED",
	"ticket:updated":  "TICKET_UPDATED",
	"order:created":   "ORDER_CREATED",
	"order:cancelled": "ORDER_CANCELLED",
}

func StringifySubject(enum Subject) (string, error) {
	protoSubj, ok := Subject_name[int32(enum.Number())]
	if !ok {
		return "", fmt.Errorf("invalid subject %v", enum)
	}
	subject, ok := protoSubjToString[protoSubj]
	if !ok {
		return "", fmt.Errorf("could not map subject %v", protoSubj)
	}
	return subject, nil
}

func SubjectifyString(subject string) (Subject, error) {
	protoSubj, ok := stringToProtoSubj[subject]
	if !ok {
		return -1, fmt.Errorf("invalid subject %v", subject)
	}
	enum, ok := Subject_value[protoSubj]
	if !ok {
		return -1, fmt.Errorf("could not map subject %v", protoSubj)
	}
	return Subject(enum), nil
}
