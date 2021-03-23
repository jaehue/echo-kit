package api

type Result struct {
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
	Error   Error       `json:"error"`
}

type ArrayResult struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
}

type ArrayResultMore struct {
	Items   interface{} `json:"items"`
	HasMore bool        `json:"hasMore"`
}

type Error struct {
	Code     int    `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
	Details  string `json:"details,omitempty"`
	err      error
	status   int
	internal bool // an internal Error must be created by New()
}
