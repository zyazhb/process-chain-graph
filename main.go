package main

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/shirou/gopsutil/process"
	"github.com/sirupsen/logrus"
)

func main() {
	// Neo4j 4.0, defaults to no TLS therefore use bolt:// or neo4j://
	// Neo4j 3.5, defaults to self-signed certificates, TLS on, therefore use bolt+ssc:// or neo4j+ssc://
	ctx := context.Background()
	dbUri := "neo4j://localhost:7687"
	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth("neo4j", os.Args[1], ""))
	if err != nil {
		panic(err)
	}
	defer driver.Close(ctx)
	item, err := insertItem(ctx, driver)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", item)
}

func insertItem(ctx context.Context, driver neo4j.DriverWithContext) (*Process, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)
	result, err := session.ExecuteWrite(ctx, createItemFn)
	if err != nil {
		return nil, err
	}
	return result.(*Process), nil
}

func createItemFn(tx neo4j.ManagedTransaction) (interface{}, error) {
	pids, _ := process.Pids()
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			logrus.Warn(err)
			continue
		}
		name, _ := p.Name()
		ppid, _ := p.Ppid()
		createTime, _ := p.CreateTime()
		cmdline, _ := p.Cmdline()
		exe, _ := p.Exe()
		records, err := tx.Run(context.Background(), "CREATE (n:Process { id:$id, name:$name, pid: $pid, ppid: $ppid, cmdline: $cmdline, Exe: $exe}) RETURN n.id, n.name",
			map[string]interface{}{
				"id":         name,
				"name":       name,
				"pid":        p.Pid,
				"ppid":       ppid,
				"createtime": createTime,
				"exe":        exe,
				"cmdline":    cmdline,
			})
		if err != nil {
			return nil, err
		}
		record, err := records.Single(context.TODO())
		if err != nil {
			return nil, err
		}
		_ = record
	}
	tx.Run(context.Background(), "MATCH (p:Process),(pp:Process) WHERE p.ppid = pp.pid CREATE (p)-[:CHILD_OF]->(pp)", nil)
	return &Process{}, nil
}

type Process struct {
	UUID    string `json:"uuid,omitempty"`
	Name    string
	Pid     int       `json:"pid,omitempty"`
	Ppid    int       `json:"ppid,omitempty"`
	Cmdline string    `json:"cmdline,omitempty"`
	Parent  []Process `json:"process,omitempty"`
	Exe     string    `json:"exe,omitempty"`
}

// MATCH (n) DETACH DELETE n
