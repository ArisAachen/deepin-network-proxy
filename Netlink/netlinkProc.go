package Netlink

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/ioutil"
	"pkg.deepin.io/lib/dbusutil"
	"strconv"
	"sync"
	"syscall"

	"pkg.deepin.io/lib/log"
)

// #include <linux/connector.h>
// #include <linux/cn_proc.h>
import "C"

var logger *log.Logger

// proc message
type ProcMessage struct {
	execPath string
	cwdPath  string
	pid      string
}

// Manager all procs
type ProcManager struct {
	// current all proc
	Procs map[string]ProcMessage

	// lock
	lock sync.Mutex

	// net_link module
	sock  int
	lAddr syscall.Sockaddr
	kAddr syscall.Sockaddr

	// signals
	signals *struct {
		ExecProc ProcMessage
		ExitProc ProcMessage
	}
}

func NewProcManager() *ProcManager {
	return &ProcManager{
		Procs: make(map[string]ProcMessage),
	}
}

func (p *ProcManager) GetInterfaceName() string {
	return BusInterface
}

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
	// select proc pid from /proc
	for _, info := range dirsInfo {
		// get proc message
		msg, err := getProcMsg(info.Name())
		if err != nil {
			logger.Debugf("get pid message failed, err: %v", err)
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
			logger.Debug("recv message is proc exec")
			event := &ExecProcEvent{}
			err = binary.Read(bytBuf, binary.LittleEndian, event)
			if err != nil {
				logger.Warningf("binary read ProcEventHeader failed, err: %v", err)
				continue
			}
			// pid equal Tgid means new proc is exec
			if event.ProcPid == event.ProcTGid {
				pid := strconv.Itoa(int(event.ProcPid))
				msg, err := getProcMsg(pid)
				if err != nil {
					logger.Debugf("pid [%s] dont include exec path", pid)
					continue
				}
				logger.Debugf("proc exec, pid [%s] exe [%s]", pid, msg.execPath)
				p.addProc(pid, msg)
			}
		// proc exit
		case C.PROC_EVENT_EXIT:
			logger.Debug("recv message is proc exit")
			event := &ExitProcEvent{}
			err = binary.Read(bytBuf, binary.LittleEndian, event)
			if err != nil {
				logger.Warningf("binary read ProcEventHeader failed, err: %v", err)
				continue
			}
			// pid equal Tgid means new proc is exec,
			// when exit, this is exactly right, when pthread_cancel or pthread_exit is called in main thread,
			// this result is not correct, but seldom program in this way
			if event.ProcessPid == event.ProcessTgid {
				pid := strconv.Itoa(int(event.ProcessPid))
				msg, err := getProcMsg(pid)
				if err != nil {
					logger.Debugf("pid [%s] dont include exec path", pid)
					continue
				}
				logger.Debugf("proc exec, pid [%s] exe [%s]", pid, msg.execPath)
				p.addProc(pid, msg)
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
	defer p.lock.Unlock()
	p.Procs[pid] = msg
}

// del proc
func (p *ProcManager) delProc(pid string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.Procs, pid)
}

func CreateProcsService() error {
	// get system bus
	service, err := dbusutil.NewSystemService()
	if err != nil {
		logger.Warningf("get system bus failed, err: %v", err)
		return err
	}
	manager := NewProcManager()
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

	// continue listen
	go func() {
		for {
			err = manager.listen()
		}
	}()

	// export bus path
	err = service.Export(BusPath, manager)
	if err != nil {
		logger.Warningf("export [%] failed, err: %v", BusPath, err)
		return err
	}

	// request service
	err = service.RequestName(BusServiceName)
	if err != nil {
		logger.Warningf("request [%s] failed, err: %v", BusServiceName, err)
		return err
	}

	return nil
}

func init() {
	logger = log.NewLogger("system/proc")
	logger.SetLogLevel(log.LevelDebug)
}
