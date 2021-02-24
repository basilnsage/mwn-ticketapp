FROM golang:alpine

WORKDIR tickets-orders
COPY . .
RUN go build -o orders .

CMD ["./orders"]
