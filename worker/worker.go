package worker

import (
  "errors"
  "github.com/pelletier/go-toml/v2"
  "io/fs"
  "io/ioutil"
  "log"
  "os"
  "path/filepath"
  "sync"
)

type MirrorConfigStruct struct {
  PrepareExec string `toml:prepare_exec,multiline:"true"`
  Exec string `toml:exec,multiline:"true"`
  SuccessExec string `toml:success_exec,multiline:"true"`
  FailExec string `toml:fail_exec,multiline:"true"`
  Period float64
}

type BaseConfigStruct struct {
  PublicPath string `toml:"public_path"`
  LogPath string `toml:"log_path"`
  MirrorsPath string `toml:"mirrors_path"`
}

type ConfigStruct struct {
  Base *BaseConfigStruct `toml:"base"`
  Mirrors []*MirrorConfigStruct `toml:"mirrors"`
}

var (
  ConfigLoad sync.Once
  Config *ConfigStruct
)

func LoadConfig() {
  ConfigLoad.Do(func() {
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
    err = filepath.WalkDir(Config.Base.MirrorsPath, func(path string, d fs.DirEntry, err error) error {
      if !d.IsDir() {
        log.Println("Found file " + d.Name() + ".")
      }
      if err != nil {
          return err
      }
      if !d.IsDir() && path[len(path) - 5:] == ".toml" {
        log.Println("Loading " + d.Name() + "...")
        //Config.Mirrors = append(Config.Mirrors, new(MirrorConfigStruct))
        mirrorConfigContent, err := ioutil.ReadFile(path)
        if err != nil {
          log.Fatal(err)
        }
        log.Println("Unmarshalling " + d.Name() + "...")
        err = toml.Unmarshal(mirrorConfigContent, Config)
        if err != nil {
          log.Fatal(err)
        }
      }
      return nil
    })
    if err != nil {
      log.Fatal(err)
    }
    log.Println("The daemon has been successfully started.")
  })
}