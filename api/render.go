package api

import (
	"errors"
	"net/http"

	"github.com/go-xorm/xorm"
	"github.com/labstack/echo"
)

func RenderFail(c echo.Context, err error) error {
	if err == nil {
		err = ErrorUnknown.New(nil)
	}

	var apiError Error
	if ok := errors.As(err, &apiError); ok {
		return &echo.HTTPError{
			Code: apiError.Status(),
			Message: Result{
				Error: apiError,
			},
			Internal: apiError,
		}
	}

	return &echo.HTTPError{
		Code: http.StatusInternalServerError,
		Message: Error{
			Message: err.Error(),
		},
		Internal: err,
	}
}

func RenderSuccess(c echo.Context, data interface{}) error {
	return RenderSuccessWithStatus(c, http.StatusOK, data)
}

func RenderSuccessWithStatus(c echo.Context, status int, data interface{}) error {
	req := c.Request()
	if req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" {
		dbSessionValue := req.Context().Value("DbSession")
		if dbSessionValue != nil {
			if session, ok := dbSessionValue.(*xorm.Session); ok {
				if err := session.Commit(); err != nil {
					return ErrorDB.New(err)
				}
			}
		}

	}

	return c.JSON(status, Result{
		Success: true,
		Data:    data,
	})
}
