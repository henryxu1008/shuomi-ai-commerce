package aiworkbench

// MessageNotAcceptedError proves that the submitted message did not enter execution.
// AppendUser excludes an earlier receipt before rejection. An oversized title
// skill snapshot cannot have been admitted with the supplied payload.
// Other errors with the same cause do not prove that a message was unaccepted.
type MessageNotAcceptedError struct{ Cause error }

func (e *MessageNotAcceptedError) Error() string { return e.Cause.Error() }
func (e *MessageNotAcceptedError) Unwrap() error { return e.Cause }

// ConversationNotAcceptedError proves that the serialized create path found
// no prior receipt and rejected the request before creating a conversation.
type ConversationNotAcceptedError struct{ Cause error }

func (e *ConversationNotAcceptedError) Error() string { return e.Cause.Error() }
func (e *ConversationNotAcceptedError) Unwrap() error { return e.Cause }
