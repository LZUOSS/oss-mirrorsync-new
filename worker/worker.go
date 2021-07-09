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

type MirrorConfigStruct struct {
  PrepareExec string `toml:prepare_exec,multiline:"true"`
  Exec        string `toml:exec,multiline:"true"`
  SuccessExec string `toml:success_exec,multiline:"true"`
  FailExec    string `toml:fail_exec,multiline:"true"`
  Period      float64
}

type BaseConfigStruct struct {
  PublicPath  string `toml:"public_path"`
  LogPath     string `toml:"log_path"`
  MirrorsPath string `toml:"mirrors_path"`
}

type ConfigStruct struct {
  Base    *BaseConfigStruct     `toml:"base"`
  Mirrors []*MirrorConfigStruct `toml:"mirrors"`
}

var (
  ConfigMutex sync.Mutex
  Config      *ConfigStruct
)

func LoadConfig() {
  ConfigMutex.Lock()
  log.Println("Initializing...")
  Config = new(ConfigStruct)
  log.Println("Loading config.toml...")
  baseConfigContent, err := ioutil.ReadFile("./config.toml")
  if err != nil {
    if errors.Is(err, os.ErrNotExist) {
      log.Println("./config.toml not found.")
      log.Println("Trying /etc/Chimata/config.toml...")
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
  log.Println("Loading mirrors...")
  mirrorsConfigFile, err := ioutil.ReadDir(Config.Base.MirrorsPath)
  if err != nil {
    log.Fatal(err)
  }
  for _, oneMirrorConfigFile := range mirrorsConfigFile {
    if len(oneMirrorConfigFile.Name()) > 5 && oneMirrorConfigFile.Name()[len(oneMirrorConfigFile.Name()) - 5 :] == ".toml" {
      log.Println("Loading " + oneMirrorConfigFile.Name() + "...")
      oneMirrorConfigContent, err := ioutil.ReadFile(filepath.Join(Config.Base.MirrorsPath, oneMirrorConfigFile.Name()))
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
