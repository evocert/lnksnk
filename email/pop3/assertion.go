package pop3

import "strings"

// POP3 replies as extracted from rfc1939 section 9.
const (
	OK  = "+OK"
	ERR = "-ERR"
)

// IsOK checks to see if the reply from the server contains +OK.
func IsOK(s string) bool {
	return strings.Fields(s)[0] == OK
}

// IsErr checks to see if the reply from the server contains +Err.
func IsErr(s string) bool {
	return strings.Fields(s)[0] == ERR
}
