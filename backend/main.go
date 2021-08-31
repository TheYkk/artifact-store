package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	Version = "Dev"
	DevMode = true
)

func main() {

	e := echo.New()

	if Version != "Dev" {
		e.Debug = false
		e.HideBanner = true
		e.HidePort = true
		DevMode = false
		e.Use(middleware.Logger())
	} else {
		e.Debug = true
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		e.Use(middleware.LoggerWithConfig(
			middleware.LoggerConfig{
				Format: "[${time_rfc3339}] ${status} ${method} ${path} (${remote_ip}) ${latency_human} [${id}]\n",
				Output: e.Logger.Output(),
			},
		),
		)
	}
	ctx := context.Background()
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKeyID := os.Getenv("S3_ACCESS_KEY")
	secretAccessKey := os.Getenv("S3_SECRET_KEY")
	bucketName := os.Getenv("S3_BUCKET")
	useSSL := false

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	// Create bucket if not exist
	exist, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if !exist {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	}
	// Middleware
	e.Use(
		middleware.CORS(),
		middleware.Gzip(),
		middleware.Secure(),
		middleware.Recover(),
		middleware.RequestID(),
	)

	e.GET("/ready", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	e.GET("/version", func(c echo.Context) error {
		return c.String(http.StatusOK, Version)
	})

	e.GET("/3rdparty/:url", func(c echo.Context) error {
		// get the artifact url from request url
		url := c.Param("url")
		// check file is exist in s3

		// File path is s3-bucket/domainname/filename

		// check if it's expired or not

		// pull the artifact serve and store to s3 same time

	})
	// Start server
	go func() {
		if err := e.Start(":8089"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

}
