// Copyright 2016 Jesse Allen. All rights reserved
// Released under the MIT license found in the LICENSE file.

package resource

import (
	"encoding/json"
	"errors"
	"strconv"
)

// Status represents how busy a given resource is on a scale from 0–2,
// where 0 (Free) is a completely unoccupied resource, 2 (Occupied) is
// completely occupied, and 1 (Busy) is anything between. The simplicity
// and flexibility of this scheme allows this to be used for any number
// of applications.
type Status uint8

const (
	Free     Status = iota // completely free resource
	Busy                   // resource is busy
	Occupied               // resource completely busy
)

// For the purposes of the API, it is much cleaner to keep the
// string representation to "0,1,2" instead of the pretty text.
// Use Pretty instead for those representations. Out of range
// Status values will be returned the same as Free.
func (s Status) String() string {
	return strconv.FormatUint(uint64(s.forceRange()), 10)
}

// For those few times where the pretty version of the status
// is requested, Pretty() will return the full text representation.
// Out of range status values will be returned as "Free".
func (s Status) Pretty() string {
	switch s.forceRange() {
	case Busy:
		return "Busy"
	case Occupied:
		return "Occupied"
	case Free:
		return "Free"
	default: // this should be impossible...
		return ""
	}
}

var ErrOutOfRange = errors.New("Status not in valid range")

func (s Status) inRange() bool {
	return s <= Occupied
}

// Return a valid Status in Range (only for use inside this package)
func (s Status) forceRange() Status {
	if !s.inRange() {
		return Free
	}
	return s
}

// MarshalJSON will return a numeric value in the valid range of Status values
func (s Status) MarshalJSON() ([]byte, error) {
	if !s.inRange() {
		return nil, ErrOutOfRange
	}
	return json.Marshal(uint8(s))
}

// UnmarshalJSON will assign a valid Status value from a numeric value.
func (s *Status) UnmarshalJSON(raw []byte) error {
	t := new(uint8)
	if err := json.Unmarshal(raw, t); err != nil {
		return err
	}
	*s = Status(*t)
	if !s.inRange() {
		*s = Free // set to zero value by default
		return ErrOutOfRange
	}
	return nil
}
