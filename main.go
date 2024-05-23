package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/likimiad/ozon_fintech/graph"
	"github.com/likimiad/ozon_fintech/graph/generated"
	"github.com/likimiad/ozon_fintech/internal/config"
	"github.com/likimiad/ozon_fintech/internal/database"
	"github.com/likimiad/ozon_fintech/internal/logger"
	"log/slog"
)

func main() {
	cfg := config.GetConfig()

	postService, err := database.GetDB(*cfg)
	if err != nil {
		logger.FatalError("error while making connection with database", err)
	}

	// ? GraphQL resolver
	resolver := graph.NewResolver(postService)

	// ? GraphQL server
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// ? WebSocket transport for subscriptions
	srv.AddTransport(transport.Websocket{
		Upgrader: websocket.Upgrader{
			// ! Allow all origins for WebSocket connections
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		KeepAlivePingInterval: 10 * time.Second, // ? Keep WebSocket connection alive with pings every 10 seconds
	})

	// ? Add POST transport for standard GraphQL queries and mutations
	srv.AddTransport(transport.POST{})

	http.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir("public"))))

	http.Handle("/query", srv)
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))

	slog.Info(fmt.Sprintf("connect to http://localhost:%s/ for GraphQL playground", cfg.ServerConfig.Port))
	log.Fatal(http.ListenAndServe(":"+cfg.ServerConfig.Port, nil))
}
