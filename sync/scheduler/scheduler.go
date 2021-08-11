package scheduler

import (
  "ChimataMS/worker"
  "github.com/robfig/cron"
  "log"
  "os/exec"
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

func (mirror *mirrorStruct) Run() {
  process := exec.Command(worker.Config.Base.Shell, "-c \"" + mirror.Config.Exec + "\"")
  process.Env = append(process.Env, "PUBLIC_PATH=" + worker.Config.Base.PublicPath)
  process.Dir = worker.Config.Base.PublicPath
  process.Run()
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