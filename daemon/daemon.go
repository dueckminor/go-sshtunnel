package daemon

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

type Daemon struct {
	PIDFile string
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

func (daemon Daemon) Start() {
	if process, _ := daemon.getProcessFromPIDFile(); nil != process {
		fmt.Println("Already running Daemon process ID is:", process.Pid)
		os.Exit(1)
	}
	cmd := exec.Command(os.Args[0], append([]string{"--daemon"}, os.Args[1:]...)...)
	cmd.Start()
	fmt.Println("Daemon process ID is : ", cmd.Process.Pid)
	savePID(daemon.PIDFile, cmd.Process.Pid)
	os.Exit(0)
}

func (daemon Daemon) getPIDFromFile() (pid int, err error) {
	if _, err := os.Stat(daemon.PIDFile); err != nil {
		return 0, fmt.Errorf("not running")
	}

	data, err := ioutil.ReadFile(daemon.PIDFile)
	if err != nil {
		return 0, fmt.Errorf("not running")
	}

	ProcessID, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("corrupt PID file")
	}

	return ProcessID, nil
}

func (daemon Daemon) getProcessFromPIDFile() (process *os.Process, err error) {
	ProcessID, err := daemon.getPIDFromFile()
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

func (daemon Daemon) Stop() {
	process, err := daemon.getProcessFromPIDFile()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Remove(daemon.PIDFile)
	err = process.Kill()

	if err != nil {
		fmt.Printf("Unable to kill process ID [%v] with error %v \n", process.Pid, err)
		os.Exit(1)
	} else {
		fmt.Printf("Killed process ID [%v]\n", process.Pid)
		os.Exit(0)
	}
}

func (daemon Daemon) Run(run func()) {
	// logFile, _ := os.OpenFile("/tmp/sshtunnel.log", os.O_WRONLY|os.O_CREATE|os.O_SYNC, 0755)
	// syscall.Dup2(int(logFile.Fd()), 1)
	// syscall.Dup2(int(logFile.Fd()), 2)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)

	go func() {
		signalType := <-ch
		signal.Stop(ch)
		fmt.Println("Exit command received. Exiting...")

		// this is a good place to flush everything to disk
		// before terminating.
		fmt.Println("Received signal type : ", signalType)

		// remove PID file
		os.Remove(daemon.PIDFile)

		os.Exit(0)

	}()

	run()
}

func (daemon Daemon) Main(run func()) {
	if len(os.Args) > 1 {
		if os.Args[1] == "--daemon" {
			cmd := os.Args[0]
			os.Args = os.Args[1:]
			os.Args[0] = cmd
			daemon.Run(run)
			return
		} else if os.Args[1] == "stop" {
			daemon.Stop()
			return
		}
	}
	daemon.Start()
}
