package main

import (
  "ChimataMS/scheduler"
  "ChimataMS/worker"
  "os"
  "os/signal"
  "syscall"
)

func main() {
  worker.LoadConfig()
  scheduler.InitScheduler()
  for {
    c := make(chan os.Signal)
    signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
    select {
      case <- c : {
        worker.LogFile.Close()
        os.Exit(0)
      }
    default:

    }
  }
}
