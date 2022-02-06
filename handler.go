package fibergqlgen

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gofiber/fiber/v2"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type Server struct {
	transports []graphql.Transport
	exec       *executor.Executor
}

func (s *Server) AddTransport(transport graphql.Transport) {
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

// AroundOperations is a convenience method for creating an extension that only implements operation middleware
func (s *Server) AroundOperations(f graphql.OperationMiddleware) {
	s.exec.AroundOperations(f)
}

// AroundResponses is a convenience method for creating an extension that only implements response middleware
func (s *Server) AroundResponses(f graphql.ResponseMiddleware) {
	s.exec.AroundResponses(f)
}

func NewDefaultServer(es graphql.ExecutableSchema) *Server {
	srv := New(es)

	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
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

func New(es graphql.ExecutableSchema) *Server {
	return &Server{
		exec: executor.New(es),
	}
}

func statusFor(errs gqlerror.List) int {
	switch errcode.GetErrorKind(errs) {
	case errcode.KindProtocol:
		return fiber.StatusUnprocessableEntity
	default:
		return fiber.StatusOK
	}
}

func ProcessExecution(params *graphql.RawParams, exec graphql.GraphExecutor, baseContext context.Context) ReturnSignal {
	start := graphql.Now()
	params.ReadTime = graphql.TraceTiming{Start: start, End: graphql.Now()}

	response, listOfErrors := exec.CreateOperationContext(baseContext, params)
	if listOfErrors != nil {
		resp := exec.DispatchError(graphql.WithOperationContext(baseContext, response), listOfErrors)
		return ReturnSignal{
			StatusCode: statusFor(listOfErrors),
			Response:   resp,
		}
	}
	responses, ctx := exec.DispatchOperation(baseContext, response)
	return ReturnSignal{
		StatusCode: 200,
		Response:   responses(ctx),
	}
}

type ReturnSignal struct {
	StatusCode int
	Response   *graphql.Response
}

func (s *Server) ServeGraphQL(api *fiber.Ctx) error {
	var params graphql.RawParams

	b := bytes.NewReader(api.Context().PostBody())

	decoder := json.NewDecoder(b)
	decoder.UseNumber()

	if err := decoder.Decode(&params); err != nil {
		return api.JSON(map[string]interface{}{
			"success":      false,
			"message":      "Cannot Use Request. Ensure You have provided a valid schema.",
			"returnStatus": "NOT_OK",
		})
	}

	defer func(api *fiber.Ctx) error {
		if err := recover(); err != nil {
			err := s.exec.PresentRecoveredError(api.Context(), err)
			resp := &graphql.Response{Errors: []*gqlerror.Error{err}}
			api.Status(fiber.StatusUnprocessableEntity)

			return api.JSON(resp)
		}

		return api.Next()
	}(api)

	childContext := graphql.StartOperationTrace(api.Context())
	output := ProcessExecution(&params, s.exec, childContext)
	api.Status(output.StatusCode)

	return api.JSON(output.Response)
}
