FROM golang:1.17-alpine3.14 as build
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o backend

FROM gcr.io/distroless/static:nonroot
COPY --from=build /app/backend /backend
EXPOSE 8089/tcp
USER nonroot:nonroot
ENTRYPOINT ["/backend"]