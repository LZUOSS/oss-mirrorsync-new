package main

import (
	"ChimataMS/scheduler"
	"ChimataMS/worker"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	progQuit := make(chan os.Signal)
	quitNotify := make(chan int)
	worker.LoadConfig()
	scheduler.InitScheduler(quitNotify)
	for {
		signal.Notify(progQuit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-progQuit:
			{
				os.Exit(0)
			}
		default:

		}
	}
}
