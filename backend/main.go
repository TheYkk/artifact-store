package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	netUrl "net/url"
	"os"
	"os/signal"
	"strconv"
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

	e.POST("/internal/:filename", func(c echo.Context) error {
		return nil
	})
	e.GET("/internal/:filename", func(c echo.Context) error {
		// Get file from bucketName/internal/filename
		obj, err := minioClient.GetObject(ctx, bucketName, "internal/"+c.Param("filename"), minio.GetObjectOptions{})
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		io.Copy(c.Response().Writer, obj)
		return nil
	})

	e.GET("/3rdparty/:url", func(c echo.Context) error {
		// get the artifact url from request url
		url := c.Param("url")
		// check file is exist in s3

		tags, err := minioClient.GetObjectTagging(ctx, bucketName, "3rdparty/"+url, minio.GetObjectTaggingOptions{})
		if err != nil {
			errResponse := minio.ToErrorResponse(err)
			if errResponse.Code == "NoSuchKey" {
				// Download file

				// ! I am not satisfied with these , because we add https to every url
				// ! what will happen when the artifact is only served with http?

				pUrl, _ := netUrl.Parse(url)
				pUrl.Scheme = "https"

				resp, err := http.Get(pUrl.String())
				if err != nil {
					log.Error().Err(err).Msg("Object don't have a valid expireAfter")
					return c.String(http.StatusBadRequest, "Object don't have a valid expireAfter")
				}

				// ! We are duplicate the resp.Body to achieve store to s3 and serve to http request at the same time
				var buf bytes.Buffer
				tee := io.TeeReader(resp.Body, &buf)
				io.Copy(c.Response().Writer, tee)

				// Store the artifact
				_, err = minioClient.PutObject(ctx, bucketName, "3rdparty/"+url, &buf, resp.ContentLength, minio.PutObjectOptions{
					UserTags: map[string]string{
						"etag":        resp.Header.Get("ETag"),
						"expireAfter": strconv.Itoa(int(time.Now().Add(time.Hour).Unix())),
					},
				})
				if err != nil {
					log.Error().Err(err).Msg("Object don't have a valid expireAfter")
					return c.String(http.StatusBadRequest, "Object don't have a valid expireAfter")
				}
			}
			return nil
		}

		if _, ok := tags.ToMap()["expireAfter"]; !ok {
			log.Error().Msg("Object don't have a valid tag")
			return c.String(http.StatusBadRequest, "Object don't have a valid tag")
		}
		expireAfterStr := tags.ToMap()["expireAfter"]
		expireAfter, err := strconv.Atoi(expireAfterStr)
		if err != nil {
			log.Error().Err(err).Msg("Object don't have a valid expireAfter")
			return c.String(http.StatusBadRequest, "Object don't have a valid expireAfter")

		}

		tLeftExpire := time.Since(time.Unix(int64(expireAfter), 0))
		// Which means expired
		if tLeftExpire > 0 {
			// Pull the artifact
			resp, err := http.Get(url)
			if err != nil {
				log.Error().Err(err).Msg("Object don't have a valid expireAfter")
				return c.String(http.StatusBadRequest, "Object don't have a valid expireAfter")
			}

			etag := tags.ToMap()["etag"]
			// Artifact is changed so pull it again
			if etag != resp.Header.Get("ETag") {
				// ! We can also duplicate the resp.Body to achieve store to s3 and serve to http request at the same time
				// Store the artifact
				_, err := minioClient.PutObject(ctx, bucketName, "3rdparty/"+url, resp.Body, resp.ContentLength, minio.PutObjectOptions{
					UserTags: map[string]string{
						"etag":        resp.Header.Get("ETag"),
						"expireAfter": strconv.Itoa(int(time.Now().Add(time.Hour).Unix())),
					},
				})
				if err != nil {
					log.Error().Err(err).Msg("Object don't have a valid expireAfter")
					return c.String(http.StatusBadRequest, "Object don't have a valid expireAfter")
				}
			}
			// Object is same so serve the object
		}

		obj, err := minioClient.GetObject(ctx, bucketName, "3rdparty/"+url, minio.GetObjectOptions{})
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		io.Copy(c.Response().Writer, obj)
		return nil
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
