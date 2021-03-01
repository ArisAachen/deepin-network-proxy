package Netlink

const (
	ProcDir = "/proc"
	exe     = "exe"
	cwd     = "cwd"
	cgroup  = "cgroup"
	autoPid = 0
)

const (
	BusServiceName = "com.deepin.system.procs"
	BusPath        = "/com/deepin/system/procs"
	BusInterface   = BusServiceName
)

type CbId struct {
	Idx uint32
	Val uint32
}

type CnMsg struct {
	Id   CbId
	Seq  uint32
	Ack  uint32
	Len  uint16
	Flag uint16
}

// ProcEventHeader corresponds to proc_event in cn_proc.h
type ProcEventHeader struct {
	What      uint32
	CPU       uint32
	Timestamp uint64
}

// fork proc event
type ForkProcEvent struct {
	ParentPid  uint32
	ParentTGid uint32
	ChildPid   uint32
	ChildTGid  uint32
}

// exec proc event
type ExecProcEvent struct {
	ProcPid  uint32
	ProcTGid uint32
}

// id proc event
type IdProcEvent struct {
	ProcPid  uint32
	ProcTGid uint32
}

// exit proc event
type ExitProcEvent struct {
	ProcessPid  uint32
	ProcessTgid uint32
	ExitCode    uint32
	ExitSignal  uint32
}
