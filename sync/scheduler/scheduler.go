/*
 * Copyright (c) 2021 LZUOSS
 * ChimataMS is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *  	http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 */
package scheduler

import (
	"ChimataMS/worker"
	"github.com/robfig/cron"
	"log"
	"os"
	"os/exec"
	"fmt"
	"path/filepath"
	"io"
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

func handleScript(file *os.File) {
	err := file.Close()
	if err != nil {
		log.Println("File " + file.Name() + " can't be closed: " + err.Error())
	}
	err = os.Remove(file.Name())
	if err != nil {
		log.Println("File " + file.Name() + " can't be removed " + err.Error())
	}
}

//Run interface
func (mirror *mirrorSchedulerStruct) Run() {
	runScript, err := os.CreateTemp("", "*")
	if err != nil {
		log.Println("Can't create script")
		return
	}
	fmt.Fprintln(runScript, "#!/bin/sh")
	fmt.Fprintln(runScript, mirror.Config.Exec)
	process := exec.Command("sh", runScript.Name())

	workDir := filepath.Join(worker.Config.Base.PublicPath, mirror.Config.Name)
	_, err = os.Stat(workDir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(workDir, 0755)
			if err != nil {
				log.Println("Can't make directory for" + mirror.Config.Name + ": " + err.Error())
			}
		} else {
			log.Println("Invalid directory for " + mirror.Config.Name + ": " + err.Error())
		}
	}

	var inOut io.ReadWriter
	process.Env = append(process.Env, "PUBLIC_PATH="+workDir)
	process.Dir = workDir
	process.Stdout = inOut
	process.Stderr = inOut
	process.Start()

	for process.Process == nil {
		select {
			case <-mirror.quitNotify:
			{
				err = process.Process.Kill()
				if err != nil {
					log.Println("Process " + mirror.Config.Name + " can't be stopped")
				}
				handleScript(runScript)
				return
			}
			default:
		}
	}

	state, _ := process.Process.Wait()
	if !state.Success() {
		log.Println("Mirror " + mirror.Config.Name + " failed:")
		io.Copy(os.Stderr, inOut)
		handleScript(runScript)
		return
	}
	handleScript(runScript)
}

//Crontab
var (
	mirrorScheduler *cron.Cron
)

//Initialize the scheduler
func InitScheduler(quitNotify chan int) {
	log.Println("Init scheduler...")
	worker.ConfigMutex.RLock()
	mirrorScheduler = cron.New()
	for _, mirror := range worker.Config.Mirrors {
		mirror := mirror
		go func(quitNotify chan int) {
			curMirror := new(mirrorSchedulerStruct)
			curMirror.Config = mirror

			if mirror.InitExec != "" {
				runScript, err := os.CreateTemp("", "*")
				if err != nil {
					log.Println("Can't create script")
					return
				}
				fmt.Fprintln(runScript, "#!/bin/sh")
				fmt.Fprintln(runScript, mirror.InitExec)
				init := exec.Command("sh", runScript.Name())
				workDir := filepath.Join(worker.Config.Base.PublicPath, mirror.Name)
				init.Env = append(init.Env, "PUBLIC_PATH="+workDir)
				init.Dir = workDir
				init.Stdout = os.Stdout
				init.Stderr = os.Stderr
				err = init.Start()
				if err != nil {
					log.Println("Init cannot start.")
				}
				for !init.ProcessState.Exited() {
					select {
					case <-quitNotify:
						{
							_ = init.Process.Kill()
							handleScript(runScript)
							return
						}
					}
				}
				if !init.ProcessState.Success() {
					log.Println("Init failed.")
					handleScript(runScript)
					return
				}
				handleScript(runScript)
			}

			err := mirrorScheduler.AddJob(mirror.Period, curMirror)
			if err != nil {
				log.Println("Cron can't add mirror " + mirror.Name + ": " + err.Error())
				return
			}
			mirrorSchedulerMap[mirror.Name] = curMirror

		}(quitNotify)
	}

	worker.ConfigMutex.RUnlock()
	mirrorScheduler.Run()
}
