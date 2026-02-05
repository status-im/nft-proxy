package handlers

import (
	"net/http"
	"strings"
)

func ExtractAlchemyPath(r *http.Request) string {
	path := r.URL.Path

	prefix := "/nft/v3/"
	idx := strings.Index(path, prefix)
	if idx == -1 {
		return ""
	}

	endpointPath := path[idx+len(prefix)-1:]
	return endpointPath
}
