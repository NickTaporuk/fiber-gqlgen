package transport

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	fibergqlgen "github.com/NickTaporuk/fiber-gqlgen"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// GET implements the GET side of the default HTTP transport
// defined in https://github.com/APIs-guru/graphql-over-http#get
type GET struct{}

var _ fibergqlgen.Transport = GET{}

func (h GET) Supports(c *fiber.Ctx) bool {
	if c.Get("Upgrade") != "" {
		return false
	}

	return c.Method() == "GET"
}

func (h GET) Do(c *fiber.Ctx, exec graphql.GraphExecutor) error {
	c.Set("Content-Type", "application/json")

	raw := &graphql.RawParams{
		Query:         c.Query("query"),
		OperationName: c.Query("operationName"),
	}
	raw.ReadTime.Start = graphql.Now()

	if variables := c.Query("variables"); variables != "" {
		if err := c.App().Config().JSONDecoder(utils.UnsafeBytes(variables), &raw.Variables); err != nil {
			c.Status(fiber.StatusBadRequest)
			return writeJsonError(c, "variables could not be decoded")
		}
	}

	if extensions := c.Query("extensions"); extensions != "" {
		if err := c.App().Config().JSONDecoder(utils.UnsafeBytes(extensions), &raw.Extensions); err != nil {
			c.Status(fiber.StatusBadRequest)
			return writeJsonError(c, "extensions could not be decoded")
		}
	}

	raw.ReadTime.End = graphql.Now()

	rc, err := exec.CreateOperationContext(c.Context(), raw)
	if err != nil {
		c.Status(statusFor(err))
		resp := exec.DispatchError(graphql.WithOperationContext(c.Context(), rc), err)
		return writeJson(c, resp)
	}
	op := rc.Doc.Operations.ForName(rc.OperationName)
	if op.Operation != ast.Query {
		c.Status(fiber.StatusNotAcceptable)
		return writeJsonError(c, "GET requests only allow query operations")
	}

	responses, ctx := exec.DispatchOperation(c.Context(), rc)
	return writeJson(c, responses(ctx))
}

func statusFor(errs gqlerror.List) int {
	switch errcode.GetErrorKind(errs) {
	case errcode.KindProtocol:
		return fiber.StatusUnprocessableEntity
	default:
		return fiber.StatusOK
	}
}
