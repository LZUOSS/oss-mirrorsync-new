package scheduler

import (
  "ChimataMS/worker"
  "github.com/robfig/cron"
  "log"
  "os"
  "path/filepath"
)

type mirrorStruct struct {
  Config *worker.MirrorConfigStruct
  Channel chan int
  SyncStatus string
  LastSyncTime string
}

var (
  mirrorMap map[string]*mirrorStruct
)

func (*mirrorStruct) Run() {

}

func (*mirrorStruct) UpdateRecord() {

}

var (
  mirrorCron *cron.Cron
)

func InitScheduler() {
  mirrorCron = cron.New()
  worker.ConfigMutex.RLock()
   for _, mirror := range worker.Config.Mirrors {
     curMirror := new(mirrorStruct)
     curMirror.Config = mirror
     err := mirrorCron.AddJob(mirror.Period, curMirror)
     if err != nil {
       log.Println("Cron can't load mirror " + mirror.Name + ".")
       continue
     }
     mirrorMap[mirror.Name] = curMirror
   }
   worker.ConfigMutex.RUnlock()
}