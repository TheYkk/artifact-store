package main

import (
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

var ApiUrl = "http://localhost:8089"

func main() {
	flag.StringVar(&ApiUrl, "api-url", ApiUrl, "--api-url http://artifact.com")
	flag.Parse()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Logger = log.With().Caller().Logger()

	if len(os.Args) > 2 {
		action := os.Args[1]
		filename := os.Args[2]
		switch action {
		case "upload":
			client := &http.Client{}
			data, err := os.Open(filename)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
			req, err := http.NewRequest("POST", ApiUrl+"/internal/"+filename, data)
			if err != nil {
				log.Fatal().Err(err).Send()
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
			content, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
			log.Info().Msg(string(content))
		case "get":
			client := &http.Client{}
			req, err := http.NewRequest("GET", ApiUrl+"/internal/"+filename, nil)
			if err != nil {
				log.Fatal().Err(err).Send()
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Fatal().Err(err).Send()
			}

			// Destination
			dst, err := os.Create(filename)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
			defer dst.Close()

			// Copy
			if _, err = io.Copy(dst, resp.Body); err != nil {
				log.Fatal().Err(err).Send()
			}

		}
	}
}
