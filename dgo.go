package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type CancelFunc func()

func getDgraphClient() (*dgo.Dgraph, CancelFunc) {
	conn, err := grpc.Dial("127.0.0.1:9080", grpc.WithInsecure())
	if err != nil {
		log.Fatal("While trying to dial gRPC")
	}

	dc := api.NewDgraphClient(conn)
	dg := dgo.NewDgraphClient(dc)
	// ctx := context.Background()

	// // Perform login call. If the Dgraph cluster does not have ACL and
	// // enterprise features enabled, this call should be skipped.
	// for {
	// 	// Keep retrying until we succeed or receive a non-retriable error.
	// 	err = dg.Login(ctx, "groot", "password")
	// 	if err == nil || !strings.Contains(err.Error(), "Please retry") {
	// 		break
	// 	}
	// 	time.Sleep(time.Second)
	// }
	if err != nil {
		log.Fatalf("While trying to login %v", err.Error())
	}

	return dg, func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error while closing connection:%v", err)
		}
	}
}

func SetObject(p Process) map[string]string {
	dg, cancel := getDgraphClient()
	defer cancel()

	op := &api.Operation{}
	op.Schema = `
		uuid: string @index(exact).
		pid: int .
		ppid: int .
		parent: [uid] @reverse .
		cmdline: string .
		exe: string .
		type Process {
			uuid: string
			pid: int
			ppid: int
			parent: [Process]
			cmdline: string
			exe: string
		}
	`

	ctx := context.Background()
	if err := dg.Alter(ctx, op); err != nil {
		log.Fatal(err)
	}

	mu := &api.Mutation{
		CommitNow: true,
	}
	pb, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}
	mu.SetJson = pb
	response, err := dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}
	logrus.Info(response)
	return response.Uids
}

func QueryProcess(uuid string) *Process {
	dg, cancel := getDgraphClient()
	defer cancel()
	variables := make(map[string]string)
	variables["$id"] = uuid
	const q = `query process($id: string){
		Process(func: eq(uuid,$id)) {
			uid
			uuid
			pid
			ppid
			cmdline
			exe
			dgraph.type
			}
		}`
	ctx := context.Background()
	resp, err := dg.NewTxn().QueryWithVars(ctx, q, variables)
	if err != nil {
		log.Fatal(err)
	}
	var p struct {
		Process []Process `json:"process"`
	}
	json.Unmarshal(resp.Json, &p)
	if len(p.Process) == 0 {
		return nil
	}
	return &p.Process[0]
}
