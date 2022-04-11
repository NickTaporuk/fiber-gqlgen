package transport

import (
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gofiber/fiber/v2"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// SendError sends a best effort error to a raw response writer. It assumes the client can understand the standard
// json error response
func SendError(c *fiber.Ctx, code int, errors ...*gqlerror.Error) error {
	c.Status(code)

	return c.JSON(&graphql.Response{Errors: errors})
}

// SendErrorf wraps SendError to add formatted messages
func SendErrorf(c *fiber.Ctx, code int, format string, args ...interface{}) error {
	return SendError(c, code, &gqlerror.Error{Message: fmt.Sprintf(format, args...)})
}
