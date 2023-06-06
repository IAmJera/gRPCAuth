FROM golang:1.20.4
RUN mkdir /app
RUN mkdir /data
ADD . /app
WORKDIR /app
RUN go build -o main ./server/server.go
EXPOSE 8888
ENTRYPOINT ["/app/main"]