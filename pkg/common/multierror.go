package common

import "strings"

type MultiError struct {
	Errors []error
}

func (errs MultiError) Error() string {
	var msgs []string
	for _, err := range errs.Errors {
		if err != nil {
			msgs = append(msgs, err.Error())
		}
	}
	return strings.Join(msgs, "; ")
}

func (m *MultiError) NilOrError() error {
	if len(m.Errors) == 0 {
		return nil
	}
	return m
}
