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
	"context"
	"errors"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"github.com/robfig/cron/v3"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	syncInit = iota
	syncSyncing
	syncSuccess
	syncFailed
)

//Task that includes config and sync infomation
type mirrorSchedulerStruct struct {
	Config         *worker.MirrorConfigStruct
	quitNotify     chan os.Signal
	SyncStatus     int    `toml:sync_status`
	LastChangeTime string `toml:last_change_time`
	EntryID        cron.EntryID
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

func runScript(ctx context.Context, scriptContent *string, workDir *string, err *error, status chan int) {
	var scriptFile *os.File
	scriptFile, *err = os.CreateTemp("", "*")
	if *err != nil {
		status <- syncFailed
		err2 := scriptFile.Close()
		*err = errors.New((*err).Error() + "\n" + err2.Error())
		return
	}
	defer handleScript(scriptFile)
	fmt.Fprintln(scriptFile, "#!/bin/sh")
	fmt.Fprintln(scriptFile, *scriptContent)
	cmd := exec.CommandContext(ctx, "sh", scriptFile.Name())

	cmd.Env = append(cmd.Env, "PUBLIC_PATH="+*workDir)
	cmd.Dir = *workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	*err = cmd.Run()
	select {
	case <-ctx.Done():
		{
			return
		}
	default:
		{
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
	}
}

func runHookScript(ctx context.Context, scriptContent *string, workDir *string, err *error) {
	var scriptFile *os.File
	scriptFile, *err = os.CreateTemp("", "*")
	if *err != nil {
		return
	}
	defer handleScript(scriptFile)
	fmt.Fprintln(scriptFile, "#!/bin/sh")
	fmt.Fprintln(scriptFile, *scriptContent)
	cmd := exec.CommandContext(ctx, "sh", scriptFile.Name())

	cmd.Env = append(cmd.Env, "PUBLIC_PATH="+*workDir)
	cmd.Dir = *workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	*err = cmd.Run()
}

func (mirror *mirrorSchedulerStruct) mkdir() (string, error) {
	workDir := filepath.Join(worker.Config.Base.PublicPath, mirror.Config.Name)
	dir, err := os.Stat(workDir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(workDir, 0755)
			if err != nil {
				return workDir, err
			}
		} else if !dir.IsDir() {
			return workDir, err
		}
	}
	return workDir, nil
}

func (mirror *mirrorSchedulerStruct) updateStatus(status int) {
	mirror.SyncStatus = status
	mirror.LastChangeTime = time.Now().String()
	mirrorStatusToml, _ := toml.Marshal(mirror)
	err := ioutil.WriteFile(filepath.Join(worker.Config.Base.RecordPath, mirror.Config.Name+".toml"),
		mirrorStatusToml,
		0755)
	if err != nil {
		log.Println(err)
	}
}

//Run interface
func (mirror *mirrorSchedulerStruct) Run() {
	worker.ConfigMutex.RLock()
	log.Println("Start sync mirror " + mirror.Config.Name + "...")
	go mirror.updateStatus(syncSyncing)

	workDir, err := mirror.mkdir()
	ctx, ctxCancel := context.WithCancel(context.Background())

	retStatus := make(chan int)
	go runScript(ctx, &mirror.Config.Exec, &workDir, &err, retStatus)

	select {
	case <-mirror.quitNotify:
		{
			ctxCancel()
			log.Println("Stopping mirror " + mirror.Config.Name + " sync...")
			return
		}
	case mirror.SyncStatus = <-retStatus:
		{
			if mirror.SyncStatus == syncSuccess {
				go mirror.updateStatus(syncSuccess)
				log.Println("Successfully sync mirror " + mirror.Config.Name)
				if mirror.Config.SuccessExec != "" {
					runHookScript(ctx, &mirror.Config.SuccessExec, &workDir, &err)
					if err != nil {
						log.Println("Success Hook Run Failed: " + err.Error())
					}
				}
			} else if err != nil {
				go mirror.updateStatus(syncFailed)
				log.Println("Error(s) occured: " + err.Error())
				if mirror.Config.FailExec != "" {
					runHookScript(ctx, &mirror.Config.FailExec, &workDir, &err)
					if err != nil {
						log.Println("Fail Hook Run Failed: " + err.Error())
					}
				}
			}
			ctxCancel()
		}
	}
	log.Println("Exiting mirror " + mirror.Config.Name + " sync...")
	worker.ConfigMutex.RUnlock()
}

//Crontab
var (
	mirrorScheduler *cron.Cron
)

//Initialize the scheduler
func InitScheduler() {
	log.Println("Init scheduler...")
	worker.ConfigMutex.RLock()
	mirrorScheduler = cron.New(cron.WithChain(
		cron.SkipIfStillRunning(cron.DefaultLogger),
	))
	for _, mirrorConfig := range worker.Config.Mirrors {
		mirrorConfig := mirrorConfig
		go func() {
			mirror := new(mirrorSchedulerStruct)
			mirror.Config = mirrorConfig
			mirror.quitNotify = make(chan os.Signal)
			go mirror.updateStatus(syncInit)
			if mirror.Config.InitExec != "" {

				workDir, err := mirror.mkdir()
				ctx, ctxCancel := context.WithCancel(context.Background())

				retStatus := make(chan int)
				go runScript(ctx, &mirror.Config.InitExec, &workDir, &err, retStatus)
				select {
				case <-mirror.quitNotify:
					{
						ctxCancel()
						return
					}
				case status := <-retStatus:
					{
						if status != syncSuccess {
							ctxCancel()
							return
						}
						ctxCancel()
					}
				}
			}

			var err error
			mirror.EntryID, err = mirrorScheduler.AddJob(mirror.Config.Period, mirror)
			if err != nil {
				log.Println("Cron can't add mirror " + mirror.Config.Name + ": " + err.Error())
				return
			}
			mirrorSchedulerMap[mirror.Config.Name] = mirror

		}()
	}

	worker.ConfigMutex.RUnlock()
	mirrorScheduler.Run()
}

func StopAllSync() {
	for _, mirror := range mirrorSchedulerMap {
		close(mirror.quitNotify)
	}
}
