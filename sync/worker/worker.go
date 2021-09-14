package worker

import (
	"errors"
	"github.com/pelletier/go-toml/v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type MirrorConfigStruct struct {
	Name        string `toml:"name"`
	InitExec    string `toml:"init_exec" multiline:"true"`
	Exec        string `toml:"exec" multiline:"true"`
	SuccessExec string `toml:"success_exec" multiline:"true"`
	FailExec    string `toml:"fail_exec" multiline:"true"`
	Period      string `toml:"period"`
}

type BaseConfigStruct struct {
	PublicPath       string `toml:"public_path"`
	LogPath          string `toml:"log_path"`
	MirrorConfigPath string `toml:"mirror_config_path"`
	RecordPath       string `toml:"record_path"`
	Shell            string `toml:"shell"`
}

type ConfigStruct struct {
	Base    *BaseConfigStruct     `toml:"base"`
	Mirrors []*MirrorConfigStruct `toml:"mirrors"`
}

var (
	ConfigMutex sync.RWMutex
	Config      *ConfigStruct
	LogFile     *os.File
)

func checkConfig() bool {
	if Config.Base.Shell == "" {
		Config.Base.Shell = "bash"
	}
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
	Config.Base.LogPath, err = filepath.Abs(Config.Base.LogPath)
	if err != nil {
		log.Println("The log path is not valid.")
		return false
	}
	Config.Base.RecordPath, err = filepath.Abs(Config.Base.RecordPath)
	if err != nil {
		log.Println("The record path is not valid.")
		return false
	}
	return true
}

func initLogger() {
	var err error
	LogFile, err = os.Create(filepath.Join(Config.Base.LogPath, time.Now().Local().Format("20060102")+".log"))
	if err != nil {
		log.Println(err)
	} else {
		log.SetOutput(LogFile)
	}

}

func LoadConfig() {
	ConfigMutex.Lock()
	log.Println("Initializing...")
	Config = new(ConfigStruct)
	log.Println("Loading config.toml...")
	baseConfigContent, err := ioutil.ReadFile("./config.toml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			baseConfigContent, err = ioutil.ReadFile("/etc/Chimata/config.toml")
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
	initLogger()
	ConfigMutex.Unlock()
}
