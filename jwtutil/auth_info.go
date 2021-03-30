package jwtutil

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
)

type AuthInfo struct {
	SessionId string
	UserId    int64
	Role      string
}

func GetAuthInfo(c echo.Context) (authInfo AuthInfo) {
	v := c.Get("user")
	if v == nil {
		return GetTokenInfo(c.Request().Header.Get(echo.HeaderAuthorization))
	}

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

	if role, ok := claims["role"]; ok {
		if s, ok := role.(string); ok {
			authInfo.Role = s
		}
	}

	return authInfo
}

func GetTokenInfo(token string) (info AuthInfo) {
	ss := strings.Split(token, ".")
	if len(ss) != 3 {
		return
	}

	payload, err := decodeSegment(ss[1])
	if err != nil {
		return
	}

	_ = json.Unmarshal(payload, &info)

	info.SessionId = ss[2]
	return
}

func decodeSegment(seg string) ([]byte, error) {
	if l := len(seg) % 4; l > 0 {
		seg += strings.Repeat("=", 4-l)
	}

	return base64.URLEncoding.DecodeString(seg)
}
