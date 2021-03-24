package filter

import (
	"os"
	"strings"

	"github.com/jaehue/echo-kit/api"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/minio/minio/pkg/wildcard"
)

type JWTConfig struct {
	Ignore []string
}

func JWT(config JWTConfig) echo.MiddlewareFunc {
	return middleware.JWTWithConfig(middleware.JWTConfig{
		SigningKey: []byte(os.Getenv("JWT_SECRET")),
		Skipper: func(c echo.Context) bool {
			reqMethod := c.Request().Method
			reqPath := c.Request().URL.Path
			for _, ignore := range config.Ignore {
				ss := strings.Split(ignore, " ")
				if len(ss) != 2 {
					continue
				}
				ignoreMethod, ignorePath := ss[0], ss[1]
				if ignoreMethod == reqMethod {
					if wildcard.Match(ignorePath, reqPath) {
						return true
					}
				}
			}
			return false
		},
		ErrorHandler: func(err error) error {
			var apiError api.Error
			switch err.(type) {
			case *jwt.ValidationError, jwt.ValidationError:
				apiError = api.ErrorTokenInvaild.New(err)
			case *echo.HTTPError:
				apiError = api.ErrorMissToken.New(err)
			default:
				apiError = api.ErrorMissToken.New(err)
			}

			return &echo.HTTPError{
				Code: apiError.Status(),
				Message: api.Result{
					Error: apiError,
				},
				Internal: err,
			}
		},
	})
}
