package filter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jaehue/converter"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"
	"github.com/sirupsen/logrus"
)

var (
	passwordRegex = regexp.MustCompile(`"(password|passwd)":(\s)*"(.*)"`)
)

type AccessLog struct {
	Id        int64  `json:"-"`
	SessionID string `json:"session_id,omitempty"`
	RequestID string `json:"request_id,omitempty"`

	Timestamp     time.Time `json:"timestamp,omitempty"`
	RemoteIP      string    `json:"remote_ip,omitempty"`
	Host          string    `json:"host,omitempty"`
	Uri           string    `json:"uri,omitempty"`
	Method        string    `json:"method,omitempty"`
	Path          string    `json:"path,omitempty"`
	Referer       string    `json:"referer,omitempty"`
	UserAgent     string    `json:"user_agent,omitempty"`
	Status        int       `json:"status,omitempty"`
	Latency       float64   `json:"latency,omitempty"`
	RequestLength int64     `json:"request_length,omitempty"`
	BytesSent     int64     `json:"bytes_sent,omitempty"`
	Hostname      string    `json:"hostname,omitempty"`

	Body       interface{}            `json:"body,omitempty"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Controller string                 `json:"controller,omitempty"`
	Action     string                 `json:"action,omitempty"`
	UserId     int64                  `json:"userId,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

type AccessLogWriter interface{ Write(accessLog *AccessLog) }

func AccessLogger(writer AccessLogWriter) echo.MiddlewareFunc {
	if writer == nil {
		writer = &defaultAccessLogWriter{}
	}

	hostname, err := os.Hostname()
	logrus.WithError(err).Error("Fail to get hostname")

	var echoRouter echoRouter

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			req := c.Request()

			accessLog := newAccessLog(c.Request())

			accessLog.Hostname = hostname

			var body []byte
			if shouldWriteBodyLog(req, accessLog) {
				body, _ = ioutil.ReadAll(req.Body)
				req.Body.Close()
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}

			start := time.Now()
			if err = next(c); err != nil {
				c.Error(err)
				var echoHTTPError *echo.HTTPError
				if ok := errors.As(err, &echoHTTPError); ok && echoHTTPError != nil {
					if echoHTTPError.Internal != nil {
						accessLog.Error = echoHTTPError.Internal.Error()
					} else {
						accessLog.Error = echoHTTPError.Error()
					}
				} else {
					accessLog.Error = err.Error()
				}

			}
			stop := time.Now()

			res := c.Response()

			accessLog.Status = res.Status
			accessLog.BytesSent = res.Size
			accessLog.Latency = converter.ToFixedNumber(
				float64(stop.Sub(start))/float64(time.Second),
				&converter.Setting{RoundDigit: 6, RoundStrategy: "ceil"},
			)
			accessLog.Controller, accessLog.Action = echoRouter.getControllerAndAction(c)
			if body != nil {
				body := passwordRegex.ReplaceAll(body, []byte(`"$1": "*"`))
				var bodyParam interface{}
				d := json.NewDecoder(bytes.NewBuffer(body))
				d.UseNumber()
				if err := d.Decode(&bodyParam); err == nil {
					accessLog.Body = bodyParam
				} else {
					accessLog.Body = string(body)
				}

			}

			for _, name := range c.ParamNames() {
				accessLog.Params[name] = c.Param(name)
			}

			writer.Write(accessLog)
			return
		}
	}
}

func newAccessLog(req *http.Request) *AccessLog {
	realIP := req.RemoteAddr
	if ip := req.Header.Get(HeaderXForwardedFor); ip != "" {
		realIP = strings.Split(ip, ", ")[0]
	} else if ip := req.Header.Get(HeaderXRealIP); ip != "" {
		realIP = ip
	} else {
		realIP, _, _ = net.SplitHostPort(realIP)
	}

	path := req.URL.Path
	if path == "" {
		path = "/"
	}

	requestLength, _ := strconv.ParseInt(req.Header.Get(HeaderContentLength), 10, 64)

	params := map[string]interface{}{}
	for k, v := range req.URL.Query() {
		params[k] = v[0]
	}

	requestId := req.Header.Get(HeaderXRequestID)
	if requestId == "" {
		requestId = random.String(32)
	}

	tokenInfo := GetTokenInfo(req.Header.Get(echo.HeaderAuthorization))

	c := &AccessLog{
		RequestID: requestId,
		SessionID: tokenInfo.SessionId,
		UserId:    tokenInfo.UserId,

		Timestamp:     time.Now(),
		RemoteIP:      realIP,
		Host:          req.Host,
		Uri:           req.RequestURI,
		Method:        req.Method,
		Path:          path,
		Params:        params,
		Referer:       req.Referer(),
		UserAgent:     req.UserAgent(),
		RequestLength: requestLength,
		// Controller: controller,
		// Action:     action,
	}

	return c
}

func shouldWriteBodyLog(req *http.Request, accessLog *AccessLog) bool {
	if accessLog == nil {
		return false
	}
	if req.Method != http.MethodPost &&
		req.Method != http.MethodPut &&
		req.Method != http.MethodPatch &&
		req.Method != http.MethodDelete {
		return false
	}

	contentType := req.Header.Get(echo.HeaderContentType)
	if !strings.HasPrefix(strings.ToLower(contentType), echo.MIMEApplicationJSON) {
		return false
	}

	return true

}

type echoRouter struct {
	once   sync.Once
	routes map[string]string
}

func (er *echoRouter) getControllerAndAction(c echo.Context) (controller, action string) {
	er.once.Do(func() { er.initialize(c) })

	if v := c.Get("controller"); v != nil {
		if controllerName, ok := v.(string); ok {
			controller = controllerName
		}
	}
	if v := c.Get("action"); v != nil {
		if actionName, ok := v.(string); ok {
			action = actionName
		}
	}

	if controller == "" || action == "" {
		handlerName := er.routes[fmt.Sprintf("%s+%s", c.Path(), c.Request().Method)]
		controller, action = er.convertHandlerNameToControllerAndAction(handlerName)
	}
	return
}

func (echoRouter) convertHandlerNameToControllerAndAction(handlerName string) (controller, action string) {
	handlerSplitIndex := strings.LastIndex(handlerName, ".")
	if handlerSplitIndex == -1 || handlerSplitIndex >= len(handlerName) {
		controller, action = "", handlerName
	} else {
		controller, action = handlerName[:handlerSplitIndex], handlerName[handlerSplitIndex+1:]
	}

	// 1. find this pattern: "(controller)"
	controller = controller[strings.Index(controller, "(")+1:]
	if index := strings.Index(controller, ")"); index > 0 {
		controller = controller[:index]
	}
	// 2. remove pointer symbol
	controller = strings.TrimPrefix(controller, "*")
	// 3. split by "/"
	if index := strings.LastIndex(controller, "/"); index > 0 {
		controller = controller[index+1:]
	}

	// remove function symbol
	action = strings.TrimRight(action, ")-fm")
	return
}

func (er *echoRouter) initialize(c echo.Context) {
	er.routes = make(map[string]string)
	for _, r := range c.Echo().Routes() {
		path := r.Path
		if len(path) == 0 || path[0] != '/' {
			path = "/" + path
		}
		er.routes[fmt.Sprintf("%s+%s", path, r.Method)] = r.Name
	}
}

type TokenInfo struct {
	SessionId string
	UserId    int64
}

func GetTokenInfo(token string) (info TokenInfo) {
	ss := strings.Split(token, ".")
	if len(ss) != 3 {
		return
	}

	info.SessionId = ss[2]

	payload, err := decodeSegment(ss[1])
	if err != nil {
		return
	}

	var v struct{ UserId int64 }

	_ = json.Unmarshal(payload, &v)

	info.UserId = v.UserId

	return
}

func decodeSegment(seg string) ([]byte, error) {
	if l := len(seg) % 4; l > 0 {
		seg += strings.Repeat("=", 4-l)
	}

	return base64.URLEncoding.DecodeString(seg)
}
