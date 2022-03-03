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
	"github.com/pelletier/go-toml/v2"
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
	Base    *BaseConfigStruct     `toml:"base"`
	Mirrors []*MirrorConfigStruct `toml:"mirrors"`
}

//Config Mutex and config Struct
var (
	ConfigMutex sync.RWMutex
	Config      *ConfigStruct
	ConfigPath  string
)

//Check if the paths are effective and translate them to absolute path
func checkConfig() bool {
	var err error
	ConfigMutex.RLock()
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
	ConfigMutex.RUnlock()
	return true
}
func LoadBaseConfig() {
	log.Println("Loading config.toml...")

	ConfigPath = "./config.toml"
	baseConfigContent, err := ioutil.ReadFile(ConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			ConfigPath = "/etc/chimata/config.toml"
			baseConfigContent, err = ioutil.ReadFile(ConfigPath)
			if err != nil {
				log.Fatalln(err)
			}
		} else {
			log.Fatalln(err)
		}
	}
	log.Println("Unmarshalling config.toml...")
	ConfigMutex.Lock()
	Config.Base = new(BaseConfigStruct)
	err = toml.Unmarshal(baseConfigContent, Config)
	if err != nil {
		log.Fatalln(err)
	}
	ConfigMutex.Unlock()
	if checkConfig() == false {
		log.Fatalln("The config file has error(s).")
	}
}

func LoadMirrorConfig() {
	log.Println("Loading mirrors...")
	ConfigMutex.Lock()
	Config.Mirrors = make([]*MirrorConfigStruct, 0)
	ConfigMutex.Unlock()
	mirrorsConfigFile, err := ioutil.ReadDir(Config.Base.MirrorConfigPath)
	if err != nil {
		log.Fatalln(err)
	}
	for _, oneMirrorConfigFile := range mirrorsConfigFile {
		if len(oneMirrorConfigFile.Name()) > 5 && oneMirrorConfigFile.Name()[len(oneMirrorConfigFile.Name())-5:] == ".toml" {
			log.Println("Loading " + oneMirrorConfigFile.Name() + "...")
			oneMirrorConfigContent, err := ioutil.ReadFile(filepath.Join(Config.Base.MirrorConfigPath, oneMirrorConfigFile.Name()))
			if err != nil {
				log.Fatalln(err)
			}
			log.Println("Unmarshalling " + oneMirrorConfigFile.Name() + "...")
			ConfigMutex.Lock()
			err = toml.Unmarshal(oneMirrorConfigContent, Config)
			if err != nil {
				log.Fatalln(err)
			}
			ConfigMutex.Unlock()
		}
	}
}

//Load config from toml files
func InitializeConfig() {

	log.Println("Initializing...")
	ConfigMutex.Lock()
	Config = new(ConfigStruct)
	ConfigMutex.Unlock()

	LoadBaseConfig()
	LoadMirrorConfig()
}
