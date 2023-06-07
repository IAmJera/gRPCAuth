package main

import (
	"gRPCAuth/api"
	"gRPCAuth/server/auth"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	s := grpc.NewServer()
	srv := &auth.GRPCServer{}
	api.RegisterAuthServer(s, srv)

	l, err := net.Listen("tcp", ":8888")
	if err != nil {
		log.Fatal(err)
	}
	if err = s.Serve(l); err != nil {
		log.Fatal(err)
	}
}
