package handler

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	fibergqlgen "github.com/NickTaporuk/fiber-gqlgen"
	"github.com/NickTaporuk/fiber-gqlgen/handler/transport"
	"github.com/gofiber/fiber/v2"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type Server struct {
	transports []fibergqlgen.Transport
	exec       *executor.Executor
}

func New(es graphql.ExecutableSchema) *Server {
	return &Server{
		exec: executor.New(es),
	}
}

func NewDefaultServer(es graphql.ExecutableSchema) *Server {
	srv := New(es)

	/*srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})*/
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New(1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	return srv
}

func (s *Server) AddTransport(transport fibergqlgen.Transport) {
	s.transports = append(s.transports, transport)
}

func (s *Server) SetErrorPresenter(f graphql.ErrorPresenterFunc) {
	s.exec.SetErrorPresenter(f)
}

func (s *Server) SetRecoverFunc(f graphql.RecoverFunc) {
	s.exec.SetRecoverFunc(f)
}

func (s *Server) SetQueryCache(cache graphql.Cache) {
	s.exec.SetQueryCache(cache)
}

func (s *Server) Use(extension graphql.HandlerExtension) {
	s.exec.Use(extension)
}

// AroundFields is a convenience method for creating an extension that only implements field middleware
func (s *Server) AroundFields(f graphql.FieldMiddleware) {
	s.exec.AroundFields(f)
}

// AroundRootFields is a convenience method for creating an extension that only implements field middleware
func (s *Server) AroundRootFields(f graphql.RootFieldMiddleware) {
	s.exec.AroundRootFields(f)
}

// AroundOperations is a convenience method for creating an extension that only implements operation middleware
func (s *Server) AroundOperations(f graphql.OperationMiddleware) {
	s.exec.AroundOperations(f)
}

// AroundResponses is a convenience method for creating an extension that only implements response middleware
func (s *Server) AroundResponses(f graphql.ResponseMiddleware) {
	s.exec.AroundResponses(f)
}

func (s *Server) getTransport(c *fiber.Ctx) fibergqlgen.Transport {
	for _, t := range s.transports {
		if t.Supports(c) {
			return t
		}
	}
	return nil
}

func (s *Server) ServeGraphQL(c *fiber.Ctx) error {
	var dErr error
	defer func() {
		if err := recover(); err != nil {
			err := s.exec.PresentRecoveredError(c.Context(), err)
			resp := &graphql.Response{Errors: []*gqlerror.Error{err.(*gqlerror.Error)}}
			c.Status(fiber.StatusUnprocessableEntity)

			dErr = c.JSON(resp)
		}
	}()

	graphql.StartOperationTrace(c.Context())

	transport := s.getTransport(c)
	if transport == nil {
		return sendErrorf(c, fiber.StatusBadRequest, "transport not supported")
	}

	if dErr != nil {
		return dErr
	}

	return transport.Do(c, s.exec)
}

func sendError(c *fiber.Ctx, code int, errors ...*gqlerror.Error) error {
	c.Status(code)

	return c.JSON(&graphql.Response{Errors: errors})
}

func sendErrorf(c *fiber.Ctx, code int, format string, args ...interface{}) error {
	return sendError(c, code, &gqlerror.Error{Message: fmt.Sprintf(format, args...)})
}

type OperationFunc func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler

func (r OperationFunc) ExtensionName() string {
	return "InlineOperationFunc"
}

func (r OperationFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return fmt.Errorf("OperationFunc can not be nil")
	}
	return nil
}

func (r OperationFunc) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return r(ctx, next)
}

type ResponseFunc func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response

func (r ResponseFunc) ExtensionName() string {
	return "InlineResponseFunc"
}

func (r ResponseFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return fmt.Errorf("ResponseFunc can not be nil")
	}
	return nil
}

func (r ResponseFunc) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	return r(ctx, next)
}

type FieldFunc func(ctx context.Context, next graphql.Resolver) (res interface{}, err error)

func (f FieldFunc) ExtensionName() string {
	return "InlineFieldFunc"
}

func (f FieldFunc) Validate(schema graphql.ExecutableSchema) error {
	if f == nil {
		return fmt.Errorf("FieldFunc can not be nil")
	}
	return nil
}

func (f FieldFunc) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	return f(ctx, next)
}
