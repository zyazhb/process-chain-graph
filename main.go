package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"strconv"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/sirupsen/logrus"
)

type Process struct {
	Uid     string    `json:"uid,omitempty"`
	UUID    string    `json:"uuid,omitempty"`
	Pid     int       `json:"pid,omitempty"`
	Ppid    int       `json:"ppid,omitempty"`
	Cmdline string    `json:"cmdline,omitempty"`
	Parent  []Process `json:"process,omitempty"`
	Exe     string    `json:"exe,omitempty"`
	DType   []string  `json:"dgraph.type,omitempty"`
}

func main() {
	pids, _ := process.Pids()
	for _, pid := range pids {
		GetProcessList(int(pid), "")
	}
}

func GetProcessList(pid int, parentuid string) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		logrus.Warn(err)
		return
	}
	currentctime, _ := p.CreateTime()
	currentUuid := CalcMD5(p.Pid, currentctime)
	if parentuid == "" && QueryProcess(currentUuid) != nil {
		logrus.Info("node exist Ignore... ", currentUuid)
		return
	}
	currentCmdline, _ := p.Cmdline()
	currentExe, _ := p.Exe()

	ppid, err := p.Ppid()
	if err != nil || ppid == 0 {
		logrus.Warn(err)
		return
	}
	logrus.Info("Found parent ", ppid)
	pp := Process{
		Uid:     parentuid,
		UUID:    currentUuid[:],
		Pid:     pid,
		Ppid:    int(ppid),
		Exe:     currentExe,
		Cmdline: currentCmdline,
		DType:   []string{"Process"},
	}

	parent, err := process.NewProcess(ppid)
	if err != nil {
		logrus.Warn(err)
		return
	}
	parentctime, _ := parent.CreateTime()
	parentUuid := CalcMD5(ppid, parentctime)

	if cachep := QueryProcess(parentUuid); cachep != nil {
		logrus.Info("Found exist process object ", cachep.Uid)
		pp.Parent = []Process{}
		pp.Parent = append(pp.Parent, *cachep)
		SetObject(pp)
		logrus.Info(p.Pid, parentctime, parent.Pid, "???", pp.UUID, " ", pp.Parent[0].UUID)
		return
	}
	parentCmdline, _ := parent.Cmdline()
	parentPpid, _ := parent.Ppid()
	parentExe, _ := parent.Exe()
	pp.Parent = []Process{
		{
			Uid:     "_:" + parentuid,
			UUID:    string(parentUuid[:]),
			Pid:     int(parent.Pid),
			Ppid:    int(parentPpid),
			Exe:     parentExe,
			Cmdline: parentCmdline,
			DType:   []string{"Process"},
		},
	}
	logrus.Info(p.Pid, parentctime, parent.Pid, "???", pp.UUID, " ", pp.Parent[0].UUID)
	resp := SetObject(pp)
	puid := resp[string(parentuid[:])]
	logrus.Debug("Parent uid: ", puid)
	GetProcessList(int(ppid), puid)
}

func CalcMD5(pid int32, time int64) string {
	data := bytes.NewBufferString(strconv.Itoa(int(time)) + strconv.Itoa(int(pid)))
	tmp := md5.Sum(data.Bytes())
	return hex.EncodeToString(tmp[:])
}
