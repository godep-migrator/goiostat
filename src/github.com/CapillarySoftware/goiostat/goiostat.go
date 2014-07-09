package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/CapillarySoftware/goiostat/diskStat"
	"github.com/CapillarySoftware/goiostat/ioStatTransform"
	"github.com/CapillarySoftware/goiostat/logOutput"
	"github.com/CapillarySoftware/goiostat/outputInterface"
	"github.com/CapillarySoftware/goiostat/statsOutput"
	"github.com/CapillarySoftware/goiostat/zmqOutput"
	"log"
	"os"
	"strings"
	"time"
)

/**
Go version of iostat, pull stats from proc and optionally log or send to a zeroMQ
*/

var interval = flag.Int("interval", 5, "Interval that stats should be reported.")
var outputType = flag.String("output", "stdout", "output should be one of the following types (stdout,zmq)")
var zmqUrl = flag.String("zmqUrl", "tcp://localhost:5400", "ZmqUrl valid formats (tcp://localhost:[port], ipc:///location/file.ipc)")

const linuxDiskStats = "/proc/diskstats"

func main() {
	flag.Parse()
	statsTransformChannel := make(chan *diskStat.DiskStat, 10)
	statsOutputChannel := make(chan *diskStat.ExtendedIoStats, 10)

	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)
	// signal.Notify(c, syscall.SIGTERM)
	// go func() {
	//     <-c
	//     log.Info("Caught signal, shutting down")
	//     close(statsTransformChannel)
	//     close(statsOutputChannel)
	//     log.Info("Shutdown complete")
	//     os.Exit(0)
	// }()
	var output outputInterface.Output
	switch *outputType {
	case "stdout":
		output = &logOutput.LogOutput{}
	case "zmq":
		zmq := &zmqOutput.ZmqOutput{}
		zmq.Connect(*zmqUrl)
		defer zmq.Close()
		output = zmq
	default:
		fmt.Println("Defaulting to stdout")
		output = &logOutput.LogOutput{}
	}

	go ioStatTransform.TransformStat(statsTransformChannel, statsOutputChannel)

	go statsOutput.Output(statsOutputChannel, output)

	for {
		readAndSendStats(statsTransformChannel)
		time.Sleep(time.Second * time.Duration(*interval))

	}
	close(statsTransformChannel)
	close(statsOutputChannel)
}

func readAndSendStats(statsTransformChannel chan *diskStat.DiskStat) {

	file, err := os.Open(linuxDiskStats)
	if nil != err {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.Fields(scanner.Text())
		stat, err := diskStat.LineToStat(line)
		if nil != err {
			log.Fatal(err)
		}
		statsTransformChannel <- &stat
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
