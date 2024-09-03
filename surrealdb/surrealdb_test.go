package aaa

import (
	"fmt"
	"testing"

	"github.com/shirou/gopsutil/process"
	"github.com/sirupsen/logrus"
	"github.com/surrealdb/surrealdb.go"
)

var db *surrealdb.DB

func InitSurrealdb() {
	// Connect to SurrealDB
	var err error
	db, err = surrealdb.New("ws://localhost:8000/rpc")
	if err != nil {
		panic(err)
	}
	if _, err = db.Use("test", "test"); err != nil {
		panic(err)
	}
}

func TestP(t *testing.T) {
	InitSurrealdb()
	pids, _ := process.Pids()
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			logrus.Warn(err)
			continue
		}
		name, _ := p.Name()
		ppid, _ := p.Ppid()
		// createTime, _ := p.CreateTime()
		cmdline, _ := p.Cmdline()
		exe, _ := p.Exe()
		_, err = db.Create("process", map[string]any{
			"id":      p.Pid,
			"name":    name,
			"pid":     p.Pid,
			"ppid":    ppid,
			"cmdline": cmdline,
			"exe":     exe,
		})
		if err != nil {
			t.Fatalf("Error creating process: %v", err)
			t.FailNow()
		}
		_, err = db.Relate(fmt.Sprintf("process:%d", ppid), "parent_of", fmt.Sprintf("process:%d", p.Pid), nil)
		if err != nil {
			t.Fatalf("Error relating process: %v", err)
			t.FailNow()
		}
	}

}
