package audio

import "errors"

var (
	ErrSilenceDetected = errors.New("silence detected")
)
