// https://github.com/jonnylangefeld/go-api/blob/c43e67c7d2d8626f1eaf724bf471272ee25b6dff/pkg/types/types.go
package httperr

import (
	"net/http"

	"github.com/go-chi/render"
)

// ErrResponse renderer type for handling all sorts of errors.
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status" example:"Resource not found."`                                         // user-level status message
	AppCode    int64  `json:"code,omitempty" example:"404"`                                                 // application-specific error code
	ErrorText  string `json:"error,omitempty" example:"The requested resource was not found on the server"` // application-level error message, for debugging
} // @name ErrorResponse

// Render implements the github.com/go-chi/render.Renderer interface for ErrResponse
func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

// ErrHTTPStatus returns a structured http response for status codes
func ErrHTTPStatus(statusCode int, err error) render.Renderer {
	var s string
	if err != nil {
		s = err.Error()
	}
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: statusCode,
		StatusText:     http.StatusText(statusCode),
		ErrorText:      s,
	}
}
