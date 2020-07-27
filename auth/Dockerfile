FROM golang:alpine

WORKDIR auth
COPY . .
RUN go build .

CMD ["./auth"]
