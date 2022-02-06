package playground

import "github.com/gofiber/fiber/v2"

// Config defines the config for middleware.
type Config struct {
	// Ttile defines default title of playground page.
	//
	// Optional. Default: Fiber GraphQL
	Title string

	// Endpoint defines query endpoint of playground page.
	//
	// Optional. Default: /query
	Endpoint string

	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next func(c *fiber.Ctx) bool
}

var ConfigDefault = Config{
	Title:    "Fiber GraphQL",
	Endpoint: "/query",
	Next:     nil,
}

func configDefault(config ...Config) Config {
	// Return default config if nothing provided
	if len(config) < 1 {
		return ConfigDefault
	}

	// Override default config
	cfg := config[0]

	// Set default values
	if cfg.Title == "" {
		cfg.Title = ConfigDefault.Title
	}

	if cfg.Endpoint == "" {
		cfg.Endpoint = ConfigDefault.Endpoint
	}

	if cfg.Next == nil {
		cfg.Next = ConfigDefault.Next
	}

	return cfg
}
