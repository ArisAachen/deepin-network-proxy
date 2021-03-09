package Netlink

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/ioutil"
	"strconv"
	"sync"
	"syscall"

	"pkg.deepin.io/lib/dbusutil"
	"pkg.deepin.io/lib/log"
)

// #include <linux/connector.h>
// #include <linux/cn_proc.h>
import "C"

var logger *log.Logger

// proc message
type ProcMessage struct {
	ExecPath    string // exe path
	Cgroup2Path string // mark cgroup v2 path
	Pid         string // Pid
}

// Manager all procs
type ProcManager struct {
	// current all proc
	Procs map[string]ProcMessage

	// lock
	lock sync.Mutex

	service *dbusutil.Service

	// net_link module
	sock  int
	lAddr syscall.Sockaddr
	kAddr syscall.Sockaddr

	//methods *struct {
	//	ChangeCGroup func() `in:"pid,cgroup" out:"err"`
	//}

	// signals
	signals *struct {
		ExecProc struct {
			execPath    string // exe path
			cgroup2Path string // mark cgroup v2 path
			pid         string // Pid
		}
		ExitProc struct {
			execPath    string // exe path
			cgroup2Path string // mark cgroup v2 path
			pid         string // Pid
		}
	}
}

func NewProcManager(service *dbusutil.Service) *ProcManager {
	return &ProcManager{
		service: service,
		Procs:   make(map[string]ProcMessage),
	}
}

func (p *ProcManager) GetInterfaceName() string {
	return BusInterface
}

//// change cgroup
//func (p *ProcManager) ChangeCGroup(pid string, cgroup string) *dbus.Error {
//	logger.Debugf("start to attach pid [%s] to cgroup [%s]", pid, cgroup)
//	// check if pid is num
//	if !com.IsPid(pid) {
//		logger.Warning("change cgroup pid is not num")
//		return dbusutil.ToError(errors.New("pid is not num"))
//	}
//	// try to get proc message
//	msg := p.getProc(pid)
//	if msg == nil {
//		logger.Warningf("pid [%s] not exist or has no exe path", pid)
//		return dbusutil.ToError(fmt.Errorf("pid [%s] not exist or has no exe path", pid))
//	}
//	err := AttachCGroup(pid, cgroup)
//	if err != nil {
//		logger.Warningf("attach pid [%s] to cgroup [%s] failed, err: %v", pid, cgroup, err)
//	}
//	logger.Debugf("attach pid [%s] to cgroup [%s] success", pid, cgroup)
//	return nil
//}

// init sock
func (p *ProcManager) initSock() error {
	var err error
	// create sock
	p.sock, err = syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM, syscall.NETLINK_CONNECTOR)
	if err != nil {
		logger.Warningf("create netlink connector failed, err: %v", err)
		return err
	}
	// init local addr
	p.lAddr = &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    autoPid,
		Groups: C.CN_IDX_PROC,
	}

	// init kernel addr
	p.kAddr = &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    0,
		Groups: C.CN_IDX_PROC,
	}

	// bind sock
	err = syscall.Bind(p.sock, p.lAddr)
	if err != nil {
		logger.Warningf("bind netlink failed, err: %v", err)
		return err
	}

	logger.Debug("init sock success")

	return nil
}

func (p *ProcManager) sendMsg(proto uint32) error {
	// message header
	cnMsg := CnMsg{
		Id: CbId{
			Idx: C.CN_IDX_PROC,
			Val: C.CN_VAL_PROC,
		},
		Ack: 0,
		Seq: 1,
		Len: uint16(binary.Size(proto)),
	}
	// msgHdr
	nlMsg := syscall.NlMsghdr{
		Len:   syscall.NLMSG_HDRLEN + uint32(binary.Size(cnMsg)+binary.Size(proto)),
		Type:  uint16(syscall.NLMSG_DONE),
		Flags: 0,
		Seq:   1,
		Pid:   uint32(syscall.Getpid()),
	}

	// write nlMsg
	buf := bytes.NewBuffer(make([]byte, 0, nlMsg.Len))
	err := binary.Write(buf, binary.LittleEndian, nlMsg)
	if err != nil {
		logger.Warningf("write nlMsg failed, err: %v", err)
		return err
	}
	// write cnMsg
	err = binary.Write(buf, binary.LittleEndian, cnMsg)
	if err != nil {
		logger.Warningf("write cnMsg failed, err: %v", err)
		return err
	}
	// write proto
	err = binary.Write(buf, binary.LittleEndian, proto)
	if err != nil {
		logger.Warningf("write proto failed, err: %v", err)
		return err
	}
	err = syscall.Sendmsg(p.sock, buf.Bytes(), nil, p.kAddr, 0)
	if err != nil {
		logger.Warningf("send message failed, err: %v", err)
		return err
	}
	logger.Debugf("send message to kernel success, proto: %v", proto)
	return nil
}

// load process in /proc
func (p *ProcManager) loadProc() error {
	// read all process from /proc
	dirsInfo, err := ioutil.ReadDir(ProcDir)
	if err != nil {
		logger.Warningf("read [%s] failed, err: %v", ProcDir, err)
		return err
	}
	// select proc Pid from /proc
	for _, info := range dirsInfo {
		// get proc message
		msg, err := getProcMsg(info.Name())
		if err != nil {
			logger.Debugf("get Pid message failed, err: %v", err)
			continue
		}
		// store process message
		p.addProc(info.Name(), msg)
	}
	return nil
}

func (p *ProcManager) listen() error {
	buf := make([]byte, 1024)
	// recv message from kernel
	nLen, _, _, _, err := syscall.Recvmsg(p.sock, buf, nil, 0)
	if err != nil {
		logger.Warningf("recv message from kernel failed, err: %v", err)
		return err
	}
	logger.Debug("success recv message from kernel")
	// check length
	if nLen < syscall.NLMSG_HDRLEN {
		logger.Warning("recv message length is less than hdr len")
		return errors.New("recv message length is less than hdr len")
	}
	// parse netlink message
	nlMsgSlice, err := syscall.ParseNetlinkMessage(buf[:nLen])
	if err != nil {
		logger.Warningf("parse netlink message failed, err: %v", err)
		return err
	}
	logger.Debug("parse netlink message success")
	// parse message
	for _, nlMsg := range nlMsgSlice {
		msg := &CnMsg{}
		header := &ProcEventHeader{}
		bytBuf := bytes.NewBuffer(nlMsg.Data)
		// read binary message
		err = binary.Read(bytBuf, binary.LittleEndian, msg)
		if err != nil {
			logger.Warningf("binary read CnMsg failed, err: %v", err)
			continue
		}
		err = binary.Read(bytBuf, binary.LittleEndian, header)
		if err != nil {
			logger.Warningf("binary read ProcEventHeader failed, err: %v", err)
			continue
		}
		switch header.What {
		// proc exec
		case C.PROC_EVENT_EXEC:
			logger.Debugf("recv message is proc exec, id: %v", C.PROC_EVENT_EXEC)
			event := &ExecProcEvent{}
			err = binary.Read(bytBuf, binary.LittleEndian, event)
			if err != nil {
				logger.Warningf("binary read ProcEventHeader failed, err: %v", err)
				continue
			}
			// Pid equal Tgid means new proc is exec
			if event.ProcPid == event.ProcTGid {
				pid := strconv.Itoa(int(event.ProcPid))
				msg, err := getProcMsg(pid)
				if err != nil {
					logger.Debugf("Pid [%s] dont include exec path", pid)
					continue
				}
				logger.Debugf("add proc exec, Pid [%s] exe [%s]", pid, msg.ExecPath)
				p.addProc(pid, msg)
			}
		// proc exit
		case C.PROC_EVENT_EXIT:
			logger.Debugf("recv message is proc exit,id :%v", C.PROC_EVENT_EXIT)
			event := &ExitProcEvent{}
			err = binary.Read(bytBuf, binary.LittleEndian, event)
			if err != nil {
				logger.Warningf("binary read ProcEventHeader failed, err: %v", err)
				continue
			}
			// Pid equal Tgid means new proc is exec,
			// when exit, this is exactly right, when pthread_cancel or pthread_exit is called in main thread,
			// this result is not correct, but seldom program in this way
			if event.ProcessPid == event.ProcessTgid {
				pid := strconv.Itoa(int(event.ProcessPid))
				logger.Debugf("del proc exec, Pid [%s]", pid)
				p.delProc(pid)
			}
		case C.PROC_EVENT_COMM:
			logger.Debugf("recv message is proc comm,id :%v", C.PROC_EVENT_COMM)
			event := &CommEvent{}
			err = binary.Read(bytBuf, binary.LittleEndian, event)
			if err != nil {
				logger.Warningf("comm event err: %v", err)
				continue
			}
			if event.ProcessPid == 17472 {
				logger.Debugf("comm vent pid %v, message: %s", event.ProcessPid, string(buf))
			}
		default:
			logger.Debugf("recv message is proc: %v", header.What)
			continue
		}
	}
	return nil
}

// add proc
func (p *ProcManager) addProc(pid string, msg ProcMessage) {
	p.lock.Lock()
	p.Procs[pid] = msg
	p.lock.Unlock()

	logger.Debugf("current exec proc %v", msg)
	err := p.service.Emit(p, "ExecProc", msg.ExecPath, msg.Cgroup2Path, msg.Pid)
	if err != nil {
		logger.Warningf("emit %v ExecProc failed, err: %v", msg, err)
		return
	}
}

// get proc
func (p *ProcManager) getProc(pid string) *ProcMessage {
	p.lock.Lock()
	msg := p.Procs[pid]
	p.lock.Unlock()
	return &msg
}

// del proc
func (p *ProcManager) delProc(pid string) {
	p.lock.Lock()
	msg, ok := p.Procs[pid]
	delete(p.Procs, pid)
	p.lock.Unlock()

	if ok {
		logger.Debugf("current exit proc %v", msg)
		err := p.service.Emit(p, "ExitProc", msg.ExecPath, msg.Cgroup2Path, msg.Pid)
		if err != nil {
			logger.Warningf("emit %v ExitProc failed, err: %v", msg, err)
			return
		}
	}
}

func CreateProcsService() error {
	// get system bus
	service, err := dbusutil.NewSystemService()
	if err != nil {
		logger.Warningf("get system bus failed, err: %v", err)
		return err
	}
	manager := NewProcManager(service)
	// init sock
	err = manager.initSock()
	if err != nil {
		logger.Warningf("init sock failed, err: %v", err)
		return err
	}
	//	send listen
	err = manager.sendMsg(C.PROC_CN_MCAST_LISTEN)
	if err != nil {
		return err
	}
	// close listen when exist
	defer func() {
		err = manager.sendMsg(C.PROC_CN_MCAST_IGNORE)
	}()

	// export bus path
	err = service.Export(BusPath, manager)
	if err != nil {
		logger.Warningf("export [%] failed, err: %v", BusPath, err)
		return err
	}

	// continue listen
	go func() {
		for {
			err = manager.listen()
		}
	}()

	err = manager.loadProc()
	if err != nil {
		logger.Warning(err)
	}

	// request service
	err = service.RequestName(BusServiceName)
	if err != nil {
		logger.Warningf("request [%s] failed, err: %v", BusServiceName, err)
		return err
	}

	service.Wait()

	return nil
}

func init() {
	logger = log.NewLogger("system/proc")
	logger.SetLogLevel(log.LevelDebug)
}
