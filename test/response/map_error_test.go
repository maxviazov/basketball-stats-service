package response_test

import (
	"errors"
	"testing"

	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
	"github.com/maxviazov/basketball-stats-service/pkg/response"
)

// fakeInvalid mimics service aggregated validation error to test mapping without reaching into internals.
type fakeInvalid struct{ fe []service.FieldError }

func (f *fakeInvalid) Error() string                { return service.ErrInvalidInput.Error() }
func (f *fakeInvalid) Unwrap() error                { return service.ErrInvalidInput }
func (f *fakeInvalid) Fields() []service.FieldError { return f.fe }

func TestMapError(t *testing.T) {
	cases := []struct {
		name     string
		in       error
		wantCode int
		wantErr  string
	}{
		{"invalid_input", &fakeInvalid{fe: []service.FieldError{{Field: "name", Message: "bad"}}}, 400, "invalid_input"},
		{"not_found", repository.ErrNotFound, 404, "not_found"},
		{"already_exists", repository.ErrAlreadyExists, 409, "already_exists"},
		{"conflict", repository.ErrConflict, 409, "conflict"},
		{"internal", errors.New("boom"), 500, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, payload := response.MapError(tc.in)
			if code != tc.wantCode || payload.Error != tc.wantErr {
				t.Fatalf("unexpected mapping: got (%d,%s) want (%d,%s)", code, payload.Error, tc.wantCode, tc.wantErr)
			}
			if tc.wantErr == "invalid_input" && len(payload.FieldErrors) == 0 {
				t.Fatalf("expected field errors in payload")
			}
		})
	}
}
