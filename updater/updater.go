/*
	Copyright (c) [2021] [LZUOSS]
	[ChimataMS] is licensed under Mulan PSL v2.
	You can use this software according to the terms and conditions of the Mulan PSL v2.
	You may obtain a copy of Mulan PSL v2 at:
			http://license.coscl.org.cn/MulanPSL2
	THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
	EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
	MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
	See the Mulan PSL v2 for more details.
*/
package updater

import (
	"github.com/fsnotify/fsnotify"
	"log"
)

var (
	mirrorWatcher *fsnotify.Watcher
	baseWatcher   *fsnotify.Watcher
)

//Initialize Updater
func InitMirrorUpdater(workDir string) (chan fsnotify.Event, chan error) {
	var err error
	mirrorWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln("Failed to initialize mirrorWatcher: " + err.Error())
	}
	err = mirrorWatcher.Add(workDir)
	if err != nil {
		log.Fatalln("Failed to add workDir to mirrorWatcher")
	}
	return mirrorWatcher.Events, mirrorWatcher.Errors
}

func CloseMirrorUpdater() {
	_ = mirrorWatcher.Close()
}

func InitBaseUpdater(configPath string) (chan fsnotify.Event, chan error) {
	var err error
	baseWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln("Failed to initialize baseWatcher: " + err.Error())
	}
	err = baseWatcher.Add(configPath)
	if err != nil {
		log.Fatalln("Failed to add configPath to baseWatcher")
	}
	return baseWatcher.Events, baseWatcher.Errors
}

func CloseBaseUpdater() {
	_ = baseWatcher.Close()
}
