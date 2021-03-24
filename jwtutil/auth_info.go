package jwtutil

import (
	"fmt"
	"strconv"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
)

type AuthInfo struct {
	UserId  int64
	IsAdmin bool
}

func GetAuthInfo(c echo.Context) (authInfo AuthInfo) {
	user := c.Get("user").(*jwt.Token)
	if user == nil {
		return
	}
	claims, ok := user.Claims.(jwt.MapClaims)
	if !ok {
		return
	}

	userId, ok := claims["userId"]
	if !ok {
		return
	}

	if f, ok := userId.(float64); ok {
		authInfo.UserId = int64(f)
	} else {
		authInfo.UserId, _ = strconv.ParseInt(fmt.Sprint(userId), 10, 64)
	}

	if authInfo.UserId == 0 {
		return
	}

	if isAdmin, ok := claims["isAdmin"]; ok {
		if b, ok := isAdmin.(bool); ok {
			authInfo.IsAdmin = b
		}
	}

	return authInfo
}
