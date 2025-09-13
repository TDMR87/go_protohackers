FROM golang:1.25 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/3_budget_chat ./
COPY internal/server ./internal/server
RUN CGO_ENABLED=0 go build -o app ./

# Run
FROM scratch
WORKDIR /app
COPY --from=build /app/app ./
EXPOSE 8080
CMD ["./app"]