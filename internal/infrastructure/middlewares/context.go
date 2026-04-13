package middlewares

import (
	"context"
)

type ContextKey string

const (
	RequestInfoContextKey ContextKey = "request_info"
	UserIDContextKey      ContextKey = "user_id"
)

func GetRequestInfo(ctx context.Context) *RequestInfo {
	requestInfo, ok := ctx.Value(RequestInfoContextKey).(*RequestInfo)

	if !ok {
		return nil
	}
	return requestInfo
}
