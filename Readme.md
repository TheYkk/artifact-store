# Readme
## How to start?
Backend is already dockerized, so we can start backend by using docker and docker compose to do that; 

``` 
docker-compose up -d
```

To start locally we need to set env variables, example of inside of backend/.env.docker. After that we can start only s3 backend with docker

``` 
docker-compose up -d s3 
```

After start the s3 we can start the backend by

```
cd backend
go mod tidy
go run .
```

The server will start at `8089` port.
## How to implement cleanup?

We can list all our object inside of bucket , if files has a `expireAfter` tag, and it's been expired we can send deletion request to s3.

Also, we can configure our s3 cleanup sequence interval.