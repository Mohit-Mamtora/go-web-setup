package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/mohit-mamtora/go-web-setup/app"
	"github.com/mohit-mamtora/go-web-setup/app/logger"
	filelogger "github.com/mohit-mamtora/go-web-setup/app/logger/filelogger"
	"github.com/mohit-mamtora/go-web-setup/app/repository"
	"github.com/mohit-mamtora/go-web-setup/app/routes"
	"github.com/mohit-mamtora/go-web-setup/app/services"
	"github.com/mohit-mamtora/go-web-setup/config"
)

var log logger.Log

func main() {

	log, err := filelogger.NewFileLogger("logs", "log.txt", 1, logger.DebugLevel, true)

	if err != nil {
		panic("logger initialization panic: " + err.Error())
	}

	defer log.Close()

	if err = godotenv.Load(); err != nil {
		panic(err)
	}

	nativeDbConnection := loadDB()

	db, err := repository.InitializeDb(nativeDbConnection, "postgres")
	if err != nil {
		log.Fatal("%v", err)
	}

	defer db.Close()

	dependencyHandler := &app.DependencyHandler{
		Logger: log,
	}

	/* Bootstrap Application */
	repo := repository.InitializeRepository(db, dependencyHandler)
	service := services.InitializeService(repo, dependencyHandler)
	server := routes.InitializeRoute(service, dependencyHandler)
	server.RegisterRoutes()

	/* Start Server  */
	go func() {
		log.Debug(config.ServerPort)
		if err = server.Start(config.ServerPort); err != nil {
			log.Fatal("%v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = server.Shutdown(ctx); err != nil {
		log.Fatal("%v", err)
	}
}

func loadDB() *sql.DB {

	/* DB connnection */
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		config.DbHost, config.DbPort, config.DbUser, config.DbPassword, config.DbName)

	nativeDbConnection, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("%e", err)
	}
	err = nativeDbConnection.Ping()
	if err != nil {
		log.Fatal("%v", err)
	}
	return nativeDbConnection
}
