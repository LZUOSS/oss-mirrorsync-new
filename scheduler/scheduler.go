package scheduler

import (
  "ChimataMS/worker"
  "github.com/robfig/cron"
  "os"
  "path/filepath"
)

type MirrorStruct struct {
  Config worker.MirrorConfigStruct
  Channel chan int
  SyncStatus string
  LastSyncTime string
}

func (*MirrorStruct) Run() {

}

func (*MirrorStruct) UpdateRecord() {

}

var (
  mirrorCron *cron.Cron
)

func InitScheduler() {
  mirrorCron = cron.New()
  worker.ConfigMutex.RLock()
   for _, mirror := range worker.Config.Mirrors {

   }
}