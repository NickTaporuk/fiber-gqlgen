package fibergqlgen

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/gofiber/fiber/v2"
)

type (
	// Transport provides support for different wire level encodings of graphql requests for Fiber, eg Form, Get, Post, Websocket
	Transport interface {
		Supports(c *fiber.Ctx) bool
		Do(c *fiber.Ctx, exec graphql.GraphExecutor) error
	}
)
