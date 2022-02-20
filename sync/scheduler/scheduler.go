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
	"ChimataMS/tools"
	"ChimataMS/worker"
	"encoding/json"
	"github.com/robfig/cron/v3"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	syncNotPrepared = iota
	syncSyncing
	syncSucceed
	syncFailed
)

//Task that includes config and sync infomation
type schedulerStruct struct {
	Config         worker.MirrorConfigStruct `json:"-"`
	quitNotify     chan struct{}
	SyncStatus     int    `json:"sync_status"`
	LastChangeTime string `json:"last_change_time"`
	//Log            string       `json:"log"`
	EntryID   cron.EntryID `json:"-"`
	IsDeleted bool         `json:"-"`
}

//Use map to save all mirror
var (
	mirrorMap = make(map[string]*schedulerStruct)
	mapMutex  sync.RWMutex
)

func (mirror *schedulerStruct) mkdir() (string, error) {
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

func (mirror *schedulerStruct) updateStatus(status int) {
	mirror.SyncStatus = status
	mirror.LastChangeTime = time.Now().String()
	mirrorStatusContext, _ := json.Marshal(mirror)
	err := ioutil.WriteFile(filepath.Join(worker.Config.Base.RecordPath, mirror.Config.Name+".json"),
		mirrorStatusContext,
		0755)
	if err != nil {
		log.Println(err)
	}
}

//Run interface
func (mirror *schedulerStruct) Run() {
	log.Println("Start sync mirror " + mirror.Config.Name + "...")
	go mirror.updateStatus(syncSyncing)

	workDir, err := mirror.mkdir()

	go tools.PromiseScript(mirror.quitNotify, &mirror.Config.Exec, &workDir, &err, func() {
		go mirror.updateStatus(syncSucceed)
		log.Println("Successfully sync mirror " + mirror.Config.Name)
		if mirror.Config.SuccessExec != "" {
			tools.PromiseScript(mirror.quitNotify, &mirror.Config.SuccessExec, &workDir, &err, func() {}, func() {})
			if err != nil {
				log.Println("Success Hook Run Failed: " + err.Error())
			}
		}
	}, func() {
		go mirror.updateStatus(syncFailed)
		log.Println("Failed to sync mirror " + mirror.Config.Name + ".")
		if mirror.Config.FailExec != "" {
			tools.PromiseScript(mirror.quitNotify, &mirror.Config.FailExec, &workDir, &err, func() {}, func() {})
			if err != nil {
				log.Println("Fail Hook Run Failed: " + err.Error())
			}
		}
	})
	if err != nil {
		log.Println("Sync mirror " + mirror.Config.Name + " Failed: " + err.Error())
	}
}

//Crontab
var (
	mirrorCron *cron.Cron
)

//Initialize the scheduler
func InitScheduler() {
	log.Println("Init scheduler...")
	mirrorCron = cron.New(cron.WithChain(
		cron.SkipIfStillRunning(cron.DefaultLogger),
	))
	worker.ConfigMutex.RLock()
	for _, mirrorConfig := range worker.Config.Mirrors {
		mirrorConfig := mirrorConfig
		go func() {
			mirror := new(schedulerStruct)
			mirror.Config = mirrorConfig
			mirror.quitNotify = make(chan struct{})
			go mirror.updateStatus(syncNotPrepared)
			if mirror.Config.InitExec != "" {
				workDir, err := mirror.mkdir()

				go tools.PromiseScript(mirror.quitNotify, &mirror.Config.InitExec, &workDir, &err, func() {
					log.Println("Initializing mirror " + mirror.Config.Name + " Successfully Ended")
				}, func() {})
				if err != nil {
					log.Println("Initializing mirror " + mirror.Config.Name + " Failed: " + err.Error())
				}
			}
			var err error
			mirror.EntryID, err = mirrorCron.AddJob(mirror.Config.Period, mirror)
			if err != nil {
				log.Println("Cron can't add mirror " + mirror.Config.Name + ": " + err.Error())
				return
			}
			mapMutex.Lock()
			mirrorMap[mirrorConfig.Name] = mirror
			mapMutex.Unlock()
		}()
	}

	worker.ConfigMutex.RUnlock()
	mirrorCron.Start()
}

func StopScheduler() {
	mapMutex.RLock()
	for _, mirror := range mirrorMap {
		close(mirror.quitNotify)
	}
	mapMutex.RUnlock()
	ctx := mirrorCron.Stop()
	<-ctx.Done()
}
