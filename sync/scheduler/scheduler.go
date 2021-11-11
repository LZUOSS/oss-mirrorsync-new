package scheduler

import (
	"ChimataMS/worker"
	"github.com/robfig/cron"
	"log"
	"os"
	"os/exec"
	"fmt"
	"path/filepath"
)

//Task that includes config and sync infomation
type mirrorSchedulerStruct struct {
	Config       *worker.MirrorConfigStruct
	Channel      chan int
	SyncStatus   string
	LastSyncTime string
	quitNotify   chan int
}

//Use map to save all mirror
var (
	mirrorSchedulerMap = make(map[string]*mirrorSchedulerStruct)
)

//Run interface
func (mirror *mirrorSchedulerStruct) Run() {
	runScript, err := os.CreateTemp("", "*")
	fmt.Fprintln(runScript, "#!/bin/sh")
	fmt.Fprintln(runScript, mirror.Config.Exec)
	process := exec.Command("sh", runScript.Name())

	workDir := filepath.Join(worker.Config.Base.PublicPath, mirror.Config.Name)
	process.Env = append(process.Env, "PUBLIC_PATH="+workDir)
	process.Dir = worker.Config.Base.PublicPath
	err = process.Run()
	if err != nil {
		log.Println("Cron can't execute mirror " + mirror.Config.Name + ".")
	}
}

//Crontab
var (
	mirrorScheduler *cron.Cron
)

//Initialize the scheduler
func InitScheduler(quitNotify chan int) {
	log.Println("Init scheduler...")
	mirrorScheduler = cron.New()
	worker.ConfigMutex.RLock()
	for _, mirror := range worker.Config.Mirrors {
		mirror := mirror
		go func(quitNotify chan int) {
			curMirror := new(mirrorSchedulerStruct)
			curMirror.Config = mirror
			if mirror.InitExec != "" {
				init := exec.Command("sh")
				init.Env = append(init.Env, "PUBLIC_PATH="+filepath.Join(worker.Config.Base.PublicPath, mirror.Name))
				init.Dir = worker.Config.Base.PublicPath
				err := init.Start()
				if err != nil {
					log.Println("Init cannot start.")
				}
				for !init.ProcessState.Exited() {
					select {
					case <-quitNotify:
						{
							_ = init.Process.Kill()
							return
						}
					}
				}
				if !init.ProcessState.Success() {
					log.Println("Init failed.")
					return
				}
			}

			err := mirrorScheduler.AddJob(mirror.Period, curMirror)
			if err != nil {
				log.Println("Cron can't add mirror " + mirror.Name + ".")
				return
			}
			mirrorSchedulerMap[mirror.Name] = curMirror

		}(quitNotify)
	}

	worker.ConfigMutex.RUnlock()
	mirrorScheduler.Run()
}
