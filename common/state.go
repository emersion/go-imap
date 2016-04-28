package common

// A connection state.
// See https://tools.ietf.org/html/rfc3501#section-3
type ConnState int

const (
	NotAuthenticatedState ConnState = iota
	AuthenticatedState
	SelectedState
	LogoutState
)
