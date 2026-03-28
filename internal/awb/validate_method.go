package awb

// Validate performs fast field-presence checks on an AWB.
// Returns the first validation error encountered; callers should
// skip the record and continue rather than aborting the batch.
func (a *AWB) Validate() error {
	if a.AWBNumber == "" {
		return ErrEmptyAWBNumber
	}
	if a.Receiver == "" {
		return ErrEmptyReceiver
	}
	if a.Address == "" {
		return ErrEmptyAddress
	}
	return nil
}