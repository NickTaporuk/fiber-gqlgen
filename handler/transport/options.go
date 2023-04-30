package transport

import (
	"github.com/99designs/gqlgen/graphql"
	fibergqlgen "github.com/NickTaporuk/fiber-gqlgen"
	"github.com/gofiber/fiber/v2"
)

// Options responds to http OPTIONS and HEAD requests
type Options struct{}

var _ fibergqlgen.Transport = Options{}

func (o Options) Supports(c *fiber.Ctx) bool {
	method := c.Method()
	return method == "HEAD" || method == "OPTIONS"
}

func (o Options) Do(c *fiber.Ctx, exec graphql.GraphExecutor) error {

	switch c.Method() {
	case fiber.MethodOptions:
		c.Set("Allow", "OPTIONS, GET, POST")
		c.Status(fiber.StatusOK)
	case fiber.MethodHead:
		c.Status(fiber.StatusMethodNotAllowed)
	}

	return nil
}
