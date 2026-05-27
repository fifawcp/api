package httpx

import (
	"net/http"
	"strconv"
)

const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

func ParsePagination(w http.ResponseWriter, r *http.Request) (page, limit int) {
	page = 1
	limit = DefaultPageLimit

	if rawPage := r.URL.Query().Get("page"); rawPage != "" {
		parsed, parseErr := strconv.Atoi(rawPage)
		if parseErr != nil {
			BadRequest(w, r, codeInvalidPageParam, errInvalidPageParam.Error())
			return
		}

		if parsed < 1 {
			BadRequest(w, r, codeInvalidPageParam, errInvalidPageParam.Error())
			return
		}

		page = parsed
	}

	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, parseErr := strconv.Atoi(rawLimit)
		if parseErr != nil {
			BadRequest(w, r, codeInvalidLimitParam, errInvalidLimitParam.Error())
			return
		}

		if parsed < 1 {
			BadRequest(w, r, codeInvalidLimitParam, errInvalidLimitParam.Error())
			return
		}

		if parsed > MaxPageLimit {
			BadRequest(w, r, codeLimitOutOfRange, errLimitOutOfRange.Error())
			return
		}

		limit = parsed
	}

	return page, limit
}
