package transport

import (
	"mime"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	fibergqlgen "github.com/efectn/fiber-gqlgen"

	"github.com/gofiber/fiber/v2"
)

// POST implements the POST side of the default HTTP transport
// defined in https://github.com/APIs-guru/graphql-over-http#post
type POST struct{}

var _ fibergqlgen.Transport = POST{}

func (h POST) Supports(c *fiber.Ctx) bool {
	if c.Get("Upgrade") != "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil {
		return false
	}

	return c.Method() == "POST" && mediaType == "application/json"
}

func (h POST) Do(c *fiber.Ctx, exec graphql.GraphExecutor) error {
	c.Set("Content-Type", "application/json")

	var params *graphql.RawParams
	start := graphql.Now()
	if err := c.App().Config().JSONDecoder(c.Body(), &params); err != nil {
		c.Status(http.StatusBadRequest)
		return writeJsonErrorf(c, "json body could not be decoded: "+err.Error())
	}
	params.ReadTime = graphql.TraceTiming{
		Start: start,
		End:   graphql.Now(),
	}

	rc, err := exec.CreateOperationContext(c.Context(), params)
	if err != nil {
		c.Status(statusFor(err))
		resp := exec.DispatchError(graphql.WithOperationContext(c.Context(), rc), err)
		return writeJson(c, resp)
	}
	responses, ctx := exec.DispatchOperation(c.Context(), rc)
	return writeJson(c, responses(ctx))
}
