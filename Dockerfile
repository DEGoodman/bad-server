FROM golang:1.13.7 AS builder
WORKDIR /go/src/github.com/degoodman/bad-server/
COPY go.mod go.sum ./
RUN go mod download
# copy source from current directory into working directory above
COPY . .
# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/degoodman/bad-server/main .
EXPOSE 8080
CMD ["./main"]  