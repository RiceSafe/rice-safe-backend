FROM golang:alpine

WORKDIR /app

RUN apk add --no-cache git make curl

# Install Air for hot reload
RUN go install github.com/air-verse/air@v1.52.3

COPY go.mod go.sum ./
RUN go mod download

CMD ["air"]
