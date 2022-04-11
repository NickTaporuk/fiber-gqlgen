package transport

import (
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gofiber/fiber/v2"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func writeJson(c *fiber.Ctx, response *graphql.Response) error {
	return c.JSON(response)
}

func writeJsonError(c *fiber.Ctx, msg string) error {
	return writeJson(c, &graphql.Response{Errors: gqlerror.List{{Message: msg}}})
}

func writeJsonErrorf(c *fiber.Ctx, format string, args ...interface{}) error {
	return writeJson(c, &graphql.Response{Errors: gqlerror.List{{Message: fmt.Sprintf(format, args...)}}})
}

func writeJsonGraphqlError(c *fiber.Ctx, err ...*gqlerror.Error) error {
	return writeJson(c, &graphql.Response{Errors: err})
}
