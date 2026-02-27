package apierror

import "net/http"

func Unauthorized(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
