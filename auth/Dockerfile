FROM golang:alpine

WORKDIR auth
COPY . .
RUN go build auth.go

CMD ["./auth"]