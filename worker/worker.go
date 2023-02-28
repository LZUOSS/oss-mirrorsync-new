package worker

import (
	"github.com/pelletier/go-toml/v2"
	"github.com/robfig/cron/v3"
)

type MirrorObject struct {
	Name        string `toml:"name"`
	Dir         string `toml:"dir"`
	InitExec    string `toml:"init_exec" multiline:"true"`
	Exec        string `toml:"exec" multiline:"true"`
	SuccessExec string `toml:"success_exec" multiline:"true"`
	FailExec    string `toml:"fail_exec" multiline:"true"`
	Period      string `toml:"period"`

	Filepath       string
	SyncStatus     int
	LastChangeTime string
	Log            string
	EntryID        cron.EntryID
}

func (*MirrorObject) Run() {

}
