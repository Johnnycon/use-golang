package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/gotesting/api/db"
	"github.com/gotesting/api/graph"
	"github.com/gotesting/api/jobs"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://chat:chat@localhost:5432/chat?sslmode=disable"
	}

	// ---- Database ----

	database, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("Connected to Postgres")

	if err := database.Migrate(ctx); err != nil {
		log.Fatalf("Failed to run app migrations: %v", err)
	}
	log.Println("App migrations complete")

	// ---- Resolver ----

	resolver := graph.NewResolver(database)

	// ---- River (job queue) ----

	// Run River's own migrations (creates river_job, river_leader, etc.).
	migrator := rivermigrate.New(riverpgxv5.New(database.Pool), nil)
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		log.Fatalf("Failed to run River migrations: %v", err)
	}
	log.Println("River migrations complete")

	// Create the worker and wire its DB + completion callback.
	worker := &jobs.ProcessMessageWorker{
		DB:         database,
		OnComplete: resolver.HandleJobComplete,
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, worker)

	riverClient, err := river.NewClient(riverpgxv5.New(database.Pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
	})
	if err != nil {
		log.Fatalf("Failed to create River client: %v", err)
	}

	if err := riverClient.Start(ctx); err != nil {
		log.Fatalf("Failed to start River: %v", err)
	}
	log.Println("River job queue started")

	// Wire the job insertion function into the resolver.
	resolver.InsertJob = func(messageID string) error {
		_, err := riverClient.Insert(ctx, jobs.ProcessMessageArgs{
			MessageID: messageID,
		}, nil)
		return err
	}

	// ---- GraphQL server ----

	srv := handler.New(graph.NewExecutableSchema(graph.Config{
		Resolvers: resolver,
	}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	})
	srv.Use(extension.Introspection{})

	mux := http.NewServeMux()
	mux.Handle("/query", corsMiddleware(srv))
	mux.Handle("/", playground.Handler("Chat API", "/query"))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	log.Println("API server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
