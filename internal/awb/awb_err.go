package awb

import "errors"

// Sentinel errors for AWB validation. Using errors.New produces comparable
// values safe for errors.Is checks without allocation on hot paths.
var (
	ErrEmptyAWBNumber = errors.New("awb: awb_number must not be empty")
	ErrEmptyReceiver  = errors.New("awb: receiver must not be empty")
	ErrEmptyAddress   = errors.New("awb: address must not be empty")
)