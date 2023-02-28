package sync

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/robfig/cron/v3"
)

type MirrorObject struct {
	Name		    string
	WorkDir     string `toml:"work_dir"`
	InitExec    string `toml:"init_exec" multiline:"true"`
	Exec        string `toml:"exec" multiline:"true"`
	SuccessExec string `toml:"success_exec" multiline:"true"`
	FailExec    string `toml:"fail_exec" multiline:"true"`
	Period      string `toml:"period"`
	LogDir		string	`toml:"log_dir`

	SyncStatus   int
	LastSyncTime string
	Cancel		context.CancelFunc
}

type DefaultConfig struct {
	WorkDirPrefix string `toml:"default_work_dir_prefix"`
	Period        string `toml:"default_period"`
	LogDir		  string `toml:"default_log_dir"`
}

const (
	syncNotPrepared = iota
	syncSyncing
	syncSucceed
	syncFailed
)

const (
	workDirErr = errors.New("WorkDir error")
	periodErr = errors.New("Period error")
	logErr = errors.New("Log error")
)

var (
	configDir     string
	defaultConfig DefaultConfig
	mirrorCron	 cron.Cron
)

func checkWorkDirAndPeriod(workDir string, period string) error {
	var rerr error
	if f, err := os.Stat(defaultConfig.WorkDirPrefix); err != nil || !f.IsDir() {
		rerr = errors.Join(rerr, workDirErr)
	}
	if _, err = cron.NewParser().Parse(defaultConfig.Period); err != nil {
		rerr = errors.Join(rerr, periodErr)
	}
	return rerr
}

func LoadWorker() {
	log.Println("Loading default config...")

	if f, err := os.Stat("./config.toml"); err == nil && !f.IsDir() {
		configDir = "./"
	} else if f, err := os.Stat("/etc/lilywhite/config.toml"); err == nil && !f.IsDir() {
		configDir = "/etc/lilywhite"
	} else {
		log.Fatalln("No configuration file")
	}

	c, err := ioutil.ReadFile(filepath.Join(configDir, "config.toml"))
	if err != nil {
		log.Fatalln(err)
	}
	err = toml.Unmarshal(defaultConfig, Config)
	if err != nil {
		log.Fatalln(err)
	}

	if err = checkWorkDirAndPeriod(defaultConfig.WorkDirPrefix, defaultConfig.Period); err != nil {
		log.Fatalln(err)
	}
	if f, err := os.Stat(defaultConfig.WorkDirPrefix); err != nil || !f.IsDir() {
		log.Fatalln(logErr)
	}

	mirrorFileList, err := ioutil.ReadDir(configDir)
	if err != nil {
		log.Fatalln(err)
	}
	
	mirrorCron = cron.New()
	for _, mirrorFile := range mirrorFileList {
		if len(mirrorFile.Name()) > 5 && mirrorFile.Name()[len(mirrorFile.Name())-5:] == ".toml" {
			go func() {
				mirrorName = mirrorFile.Name()[:-5]
				log.Println("Loading " + mirrorName + "...")
				mirrorText, err := ioutil.ReadFile(filepath.Join(ConfigDir, mirrorName)
				if err != nil {
					log.Println(err)
					return
				}

				mirror := new(MirrorObject)
				err = toml.Unmarshal(mirror, mirrorText)
				if err != nil {
					log.Println(err)
					return
				}
				if err = checkWorkDirAndPeriod(mirror.WorkDir, mirror.Period); err != nil {
					if errors.Is(err, workDirErr) {
						mirror.WorkDir = filepath.Join(defaultConfig.WorkDirPrefix, mirror.Name)
					}
					if errors.Is(err, periodErr) {
						mirror.Period = defaultConfig.Period	
					}
				}
				if f, err := os.Stat(defaultConfig.WorkDirPrefix); err != nil || f.IsDir() {
					mirror.LogDir = defaultConfig.LogDir
				}
				
				mirrorCron.AddJob(mirror)
			}
		}
	}
}

func Start() {
	mirrorCron.Start()
}

func (mirror *MirrorObject) updateStatus(status int) {
	mirror.SyncStatus = status
}

func (mirror *MirrorObject) Run() {
	log.Println("Start sync mirror " + mirror.Name + "...")
	mirror.updateStatus(syncSyncing)
}
