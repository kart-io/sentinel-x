package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	v1 "github.com/kart-io/sentinel-x/example/server/user-center/api/v1"
)

func main() {
	conn, err := grpc.NewClient("localhost:9098", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}()

	c := v1.NewUserServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.GetUser(ctx, &v1.UserRequest{Id: "1"})
	if err != nil {
		log.Fatalf("could not get user: %v", err)
	}

	fmt.Printf("User: %s, Role: %s\n", r.Username, r.Role)
}
