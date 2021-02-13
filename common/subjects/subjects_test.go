package subjects

import "testing"

func TestSubjects(t *testing.T) {
	tests := map[string]struct{
		subj Subject
		want string
	}{
		"test ticket created": {
			Subject_TICKET_CREATED,
			"ticket:created",
		},
		"test ticket updated": {
			Subject_TICKET_UPDATED,
			"ticket:updated",
		},
		"test order created": {
			Subject_ORDER_CREATED,
			"order:created",
		},
		"test order cancelled": {
			Subject_ORDER_CANCELLED,
			"order:cancelled",
		},
	}

	for name, test := range tests {
		t.Run(name, func(tester *testing.T) {
			got, err := StringifySubject(test.subj)
			if err != nil {
				tester.Fatalf("error stringifying subject: %v", err)
			}
			if got != test.want {
				tester.Fatalf("incorrect subject string: %v, want %v", got, test.want)
			}
		})
	}
}

func TestSubjectStrings(t *testing.T) {
	tests := map[string]struct{
		subj string
		want Subject
	}{
		"test ticket created": {
			"ticket:created",
			Subject_TICKET_CREATED,
		},
		"test ticket updated": {
			"ticket:updated",
			Subject_TICKET_UPDATED,
		},
		"test order created": {
			"order:created",
			Subject_ORDER_CREATED,
		},
		"test order cancelled": {
			"order:cancelled",
			//Subject_ORDER_CANCELLED,
			Subject_UNKNOWN_SUBJECT,
		},
	}

	for name, test := range tests {
		t.Run(name, func(tester *testing.T) {
			got, err := SubjectifyString(test.subj)
			if err != nil {
				tester.Fatalf("error subjectifying string: %v", err)
			}
			if got != test.want {
				tester.Fatalf("incorrect subject: %v, want %v", got, test.want)
			}
		})
	}
}