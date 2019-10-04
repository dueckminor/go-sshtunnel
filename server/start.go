package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

var (
	pidFile = "/tmp/sshtunnel.pid"
)

func start(verbose bool) {
	process, _ := getProcessFromPIDFile(pidFile)
	if nil != process {
		if verbose {
			fmt.Println("Daemon process ID is:", process.Pid, "(already running)")
		}
		return
	}

	cmd := exec.Command(os.Args[0], "daemon")
	cmd.Start()
	if verbose {
		fmt.Println("Daemon process ID is:", cmd.Process.Pid)
	}

	savePID(pidFile, cmd.Process.Pid)
}

func Start() {
	start(true)
}

func Stop() {
	process, _ := getProcessFromPIDFile(pidFile)
	if nil == process {
		fmt.Println("Daemon process is already stopped")
		return
	}

	os.Remove(pidFile)
	err := process.Kill()

	if err != nil {
		fmt.Printf("Unable to kill process ID [%v] with error %v \n", process.Pid, err)
		return
	} else {
		fmt.Printf("Killed process ID [%v]\n", process.Pid)
		return
	}
}

func savePID(PIDFile string, pid int) {
	file, err := os.Create(PIDFile)
	if err != nil {
		log.Printf("Unable to create pid file : %v\n", err)
		os.Exit(1)
	}

	defer file.Close()

	_, err = file.WriteString(strconv.Itoa(pid))

	if err != nil {
		log.Printf("Unable to create pid file : %v\n", err)
		os.Exit(1)
	}

	file.Sync() // flush to disk
}

func getPIDFromFile(PIDFile string) (pid int, err error) {
	if _, err := os.Stat(PIDFile); err != nil {
		return 0, fmt.Errorf("not running")
	}

	data, err := ioutil.ReadFile(PIDFile)
	if err != nil {
		return 0, fmt.Errorf("not running")
	}

	ProcessID, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("corrupt PID file")
	}

	return ProcessID, nil
}

func getProcessFromPIDFile(PIDFile string) (process *os.Process, err error) {
	ProcessID, err := getPIDFromFile(PIDFile)
	if err != nil {
		return nil, err
	}
	process, err = os.FindProcess(ProcessID)
	if err != nil {
		return nil, fmt.Errorf("unable to find process ID [%v] with error %v", ProcessID, err)
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return nil, err
	}

	return process, nil
}
