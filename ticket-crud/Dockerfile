FROM golang:alpine

WORKDIR ticket-crud
COPY . .
RUN go build -o crud .

CMD ["./crud"]
