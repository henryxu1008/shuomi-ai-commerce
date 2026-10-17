package aiworkbench

// MessageNotAcceptedError is produced only by the serialized AppendUser
// preflight, after excluding a prior receipt and before creating a message.
// Other errors with the same cause do not prove that a message was unaccepted.
type MessageNotAcceptedError struct{ Cause error }

func (e *MessageNotAcceptedError) Error() string { return e.Cause.Error() }
func (e *MessageNotAcceptedError) Unwrap() error { return e.Cause }
