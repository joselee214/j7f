package mq

import "errors"

// ErrNotConnected is returned when a publish command is made
// against a Producer that is not connected
var ErrNotConnected = errors.New("not connected")

// ErrStopped is returned when a publish command is
// made against a Producer that has been stopped
var ErrStopped = errors.New("stopped")

// ErrClosing is returned when a connection is closing
var ErrClosing = errors.New("closing")

// ErrAlreadyConnected is returned from ConnectToNSQD when already connected
var ErrAlreadyConnected = errors.New("already connected")

// ErrOverMaxInFlight is returned from Consumer if over max-in-flight
var ErrOverMaxInFlight = errors.New("over configure max-inflight")
