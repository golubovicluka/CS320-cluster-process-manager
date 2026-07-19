package domain

import "errors"

var (
	ErrNodeNotFound           = errors.New("node not found")
	ErrProcessNotFound        = errors.New("process not found")
	ErrDuplicateNode          = errors.New("node already exists")
	ErrDuplicateProcess       = errors.New("process already exists")
	ErrInsufficientResources  = errors.New("insufficient resources")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrNoSchedulableNode      = errors.New("no schedulable node")
	ErrSimulationNotRunning   = errors.New("simulation is not running")
	ErrMaxRestartsExceeded    = errors.New("maximum restart count exceeded")
	ErrInvalidInput           = errors.New("invalid input")
)
