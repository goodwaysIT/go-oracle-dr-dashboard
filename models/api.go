package models

type ApiResponse struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Timestamp int64     `json:"timestamp"`
} 