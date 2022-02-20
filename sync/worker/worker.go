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
package worker

import (
	"errors"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
)

//Basic config for every mirror task
type MirrorConfigStruct struct {
	Name        string `toml:"name"`
	InitExec    string `toml:"init_exec" multiline:"true"`
	Exec        string `toml:"exec" multiline:"true"`
	SuccessExec string `toml:"success_exec" multiline:"true"`
	FailExec    string `toml:"fail_exec" multiline:"true"`
	Period      string `toml:"period"`
}

//Basic config for the whole program
type BaseConfigStruct struct {
	PublicPath       string `toml:"public_path"`
	LogPath          string `toml:"log_path"`
	MirrorConfigPath string `toml:"mirror_config_path"`
	RecordPath       string `toml:"record_path"`
}

//Config include the two above
type ConfigStruct struct {
	Base    *BaseConfigStruct    `toml:"base"`
	Mirrors []MirrorConfigStruct `toml:"mirrors"`
}

//Config Mutex and config Struct
var (
	ConfigMutex sync.RWMutex
	Config      *ConfigStruct
)

//Check if the paths are effective and translate them to absolute path
func checkConfig() bool {
	var err error
	Config.Base.PublicPath, err = filepath.Abs(Config.Base.PublicPath)
	if err != nil {
		return false
	}
	if err != nil {
		log.Println("The public path is not valid.")
		return false
	}
	Config.Base.MirrorConfigPath, err = filepath.Abs(Config.Base.MirrorConfigPath)
	if err != nil {
		log.Println("The mirror config path is not valid.")
		return false
	}
	Config.Base.RecordPath, err = filepath.Abs(Config.Base.RecordPath)
	if err != nil {
		log.Println("The record path is not valid.")
		return false
	}
	return true
}

//Load config from toml files
func LoadConfig() {
	ConfigMutex.Lock()
	log.Println("Initializing...")
	Config = new(ConfigStruct)
	log.Println("Loading config.toml...")
	baseConfigContent, err := ioutil.ReadFile("./config.toml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			baseConfigContent, err = ioutil.ReadFile("/etc/chimata/config.toml")
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}
	log.Println("Unmarshalling config.toml...")
	err = toml.Unmarshal(baseConfigContent, Config)
	if err != nil {
		log.Fatal(err)
	}
	if checkConfig() == false {
		log.Fatal("The config file has error(s).")
	}
	log.Println("Loading mirrors...")
	mirrorsConfigFile, err := ioutil.ReadDir(Config.Base.MirrorConfigPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, oneMirrorConfigFile := range mirrorsConfigFile {
		if len(oneMirrorConfigFile.Name()) > 5 && oneMirrorConfigFile.Name()[len(oneMirrorConfigFile.Name())-5:] == ".toml" {
			log.Println("Loading " + oneMirrorConfigFile.Name() + "...")
			oneMirrorConfigContent, err := ioutil.ReadFile(filepath.Join(Config.Base.MirrorConfigPath, oneMirrorConfigFile.Name()))
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Unmarshalling " + oneMirrorConfigFile.Name() + "...")
			err = toml.Unmarshal(oneMirrorConfigContent, Config)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	ConfigMutex.Unlock()
}
