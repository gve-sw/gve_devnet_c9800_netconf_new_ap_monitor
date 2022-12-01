FROM golang:buster

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o ./app/netconf-ap-monitor .

CMD ["./app/netconf-ap-monitor"]