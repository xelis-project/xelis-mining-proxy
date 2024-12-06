package stratum

import "encoding/json"

type RequestIn struct {
	Id     uint32          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}
type RequestOut struct {
	Id     uint32 `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type ResponseIn struct {
	Id     uint32 `json:"id"`
	Result any    `json:"result"`
	Error  *Error `json:"error,omitempty"`
}
type ResponseOut struct {
	Id     uint32 `json:"id"`
	Result any    `json:"result"`
	Error  *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
