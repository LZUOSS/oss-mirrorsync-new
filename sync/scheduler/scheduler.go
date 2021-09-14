package scheduler

import (
	"ChimataMS/worker"
	"github.com/robfig/cron"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type mirrorStruct struct {
	Config       *worker.MirrorConfigStruct
	Channel      chan int
	SyncStatus   string
	LastSyncTime string
}

var (
	mirrorMap = make(map[string]*mirrorStruct)
)

func (mirror *mirrorStruct) Run() {
	process := exec.Command(worker.Config.Base.Shell, "-c \""+mirror.Config.Exec+"\"")
	process.Env = append(process.Env, "PUBLIC_PATH="+filepath.Join(worker.Config.Base.PublicPath, mirror.Config.Name))
	process.Dir = worker.Config.Base.PublicPath
	process.Stdout = worker.LogFile
	process.Stderr = worker.LogFile
	process.Run()
}

var (
	mirrorCron *cron.Cron
)

func InitScheduler(quitNotify chan int) {
	log.Println("Init scheduler...")
	mirrorCron = cron.New()
	worker.ConfigMutex.RLock()
	for _, mirror := range worker.Config.Mirrors {
		mirror := mirror
		go func(quitNotify chan int) {
			curMirror := new(mirrorStruct)
			curMirror.Config = mirror
			init := exec.Command(worker.Config.Base.Shell, "-c \""+mirror.InitExec+"\"")
			init.Env = append(process.Env, "PUBLIC_PATH="+filepath.Join(worker.Config.Base.PublicPath, mirror.Config.Name))
			init.Dir = worker.Config.Base.PublicPath
			init.Stdout = worker.LogFile
			init.Stderr = worker.LogFile
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
			} else {
				err = mirrorCron.AddJob(mirror.Period, curMirror)
				if err != nil {
					log.Println("Cron can't add mirror " + mirror.Name + ".")
					return
				}
				mirrorMap[mirror.Name] = curMirror
			}
		}(quitNotify)
	}

	mirrorCron.AddFunc("@daily", func() {
		worker.ConfigMutex.RLock()
		err := worker.LogFile.Close()
		if err != nil {
			log.Println(err)
		}
		worker.LogFile, err = os.Create(filepath.Join(worker.Config.Base.LogPath, time.Now().Local().Format("20060102"), ".log"))
		if err != nil {
			log.Println(err)
		} else {
			worker.ConfigMutex.RUnlock()
		}
	})

	worker.ConfigMutex.RUnlock()
	mirrorCron.Run()
}
