package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/pangpanglabs/goutils/test"
)

func TestErrorsTest(t *testing.T) {
	t.Run("Internal", func(t *testing.T) {
		err := errors.New("invalid sql")
		err = ErrorUnknown.New(err)

		test.Equals(t, err.Error(), "invalid sql")

		var apiError Error
		ok := errors.As(err, &apiError)
		test.Equals(t, ok, true)
		test.Equals(t, apiError.Code, ErrorUnknown.Code)
		test.Equals(t, apiError.Message, ErrorUnknown.Message)
		test.Equals(t, apiError.Details, "invalid sql")

		// Test do not cover
		err = ErrorDB.New(err)
		ok = errors.As(err, &apiError)
		test.Equals(t, ok, true)
		test.Equals(t, apiError.Code, ErrorUnknown.Code)
		test.Equals(t, apiError.Message, ErrorUnknown.Message)
		test.Equals(t, apiError.Details, "invalid sql")
	})

	t.Run("External", func(t *testing.T) {
		SetErrorMessagePrefix("serviceB")
		v := Error{
			Code:    ErrorDB.Code,
			Message: ErrorDB.Message,
			Details: "serviceA: invalid sql",
		}

		err := error(v)
		err = ErrorRemoteService.New(err)

		test.Equals(t, err.Error(), "serviceB: serviceA: invalid sql")

		var apiError Error
		ok := errors.As(err, &apiError)
		test.Equals(t, ok, true)
		test.Equals(t, apiError.Code, ErrorRemoteService.Code)
		test.Equals(t, apiError.Message, ErrorRemoteService.Message)
		test.Equals(t, apiError.Details, "serviceB: serviceA: invalid sql")
	})

	t.Run("err-nil", func(t *testing.T) {
		SetErrorMessagePrefix("serviceC")
		err := ErrorUnknown.New(nil)
		test.Equals(t, "serviceC: Unknown error", err.Error())
	})
	t.Run("err-custom", func(t *testing.T) {
		expError := ErrorTemplate{
			Code:    20001,
			Message: "Task exists in task group",
		}
		expStatus := http.StatusBadRequest

		SetErrorMessagePrefix("serviceD")
		customErr := errors.New("There is a task in the task group, please delete the task in the task group first")
		template := NewTemplate(20001, "Task exists in task group", http.StatusBadRequest)
		actError := template.New(customErr)
		test.Equals(t, expError.Code, actError.Code)
		test.Equals(t, expError.Message, actError.Message)
		test.Equals(t, expStatus, actError.Status())
		test.Equals(t, fmt.Sprintf("serviceD: %v", customErr), actError.Error())
	})
}
