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
	"sync"
	"time"
	"context"
)

const (
	syncInit = iota
	syncRunning
	syncSuccess
	syncFailed
)

//Task that includes config and sync infomation
type mirrorSchedulerStruct struct {
	Config       	*worker.MirrorConfigStruct
	SyncStatus   	chan int
	ctx				*context.Context
	LastSyncTime	string
	syncMutex		sync.Mutex
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

func runScript(scriptContent string, workDir string, err *error, ctx context.Context, status chan int) {
	var scriptFile *os.File
	scriptFile, *err = os.CreateTemp("", "*")
	if *err != nil {
		status <- syncFailed
		return
	}
	defer handleScript(scriptFile)
	fmt.Fprintln(scriptFile, "#!/bin/sh")
	fmt.Fprintln(scriptFile, scriptContent)
	cmd := exec.CommandContext(ctx, "sh", scriptFile.Name())

	cmd.Env = append(cmd.Env, "PUBLIC_PATH="+workDir)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	*err = cmd.Run()
	select {
		case <- ctx.Done(): {

		}
	}

	if *err != nil {
		status <- syncFailed
		return
	}
	if cmd.ProcessState.Success() {
		status <- syncSuccess
	} else {
		status <- syncFailed
	}
}

//Run interface
func (mirror *mirrorSchedulerStruct) Run() {
	mirror.syncMutex.Lock()
	log.Println("Start sync mirror " + mirror.Config.Name + "...")

	workDir := filepath.Join(worker.Config.Base.PublicPath, mirror.Config.Name)
	dir, err := os.Stat(workDir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(workDir, 0755)
			if err != nil {
				log.Println("Can't make directory for" + mirror.Config.Name + ": " + err.Error())
				return
			}
		} else if !dir.IsDir() {
			log.Println("Invalid directory for " + mirror.Config.Name + ": " + err.Error())
			return
		}
	}

	var cmd **exec.Cmd
	ctx, ctxCancel := context.WithCancel(context.Background())
	go runScript(mirror.Config.Exec, workDir, &err, ctx, mirror.SyncStatus)
	select {
		case <- mirror.quitNotify: {

		}
	}
	for cmd == nil {
		time.Sleep(1 * time.Second)
		log.Println("test")
	}
	mirror.syncMutex.Unlock()
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
