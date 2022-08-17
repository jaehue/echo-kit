package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"xorm.io/xorm"
)

func RenderFail(c echo.Context, err error) error {
	if err == nil {
		err = ErrorUnknown.New(nil)
	}

	var apiError Error
	if ok := errors.As(err, &apiError); !ok {
		apiError = ErrorUnknown.New(err)

	}

	return &echo.HTTPError{
		Code: apiError.Status(),
		Message: Result{
			Error: apiError,
		},
		Internal: apiError,
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
