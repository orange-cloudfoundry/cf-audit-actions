package main

import (
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalFlag(value string) error {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}
