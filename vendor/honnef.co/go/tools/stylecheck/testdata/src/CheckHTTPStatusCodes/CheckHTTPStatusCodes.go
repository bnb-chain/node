// Package pkg ...
package pkg

import "net/http"

func fn() {
	// Check all the supported functions
	http.Error(nil, "", 506)         // MATCH "http.StatusVariantAlsoNegotiates"
	http.Redirect(nil, nil, "", 506) // MATCH "http.StatusVariantAlsoNegotiates"
	http.StatusText(506)             // MATCH "http.StatusVariantAlsoNegotiates"
	http.RedirectHandler("", 506)    // MATCH "http.StatusVariantAlsoNegotiates"

	// Don't flag literals with no known constant
	http.StatusText(600)

	// Don't flag constants
	http.StatusText(http.StatusAccepted)

	// Don't flag items on the whitelist (well known codes)
	http.StatusText(404)
}
