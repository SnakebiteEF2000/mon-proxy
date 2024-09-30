package logger

import (
	"os"

	"github.com/op/go-logging"
)

var Log = logging.MustGetLogger("monProxyGlobal")

var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

func setupLevel(debug bool) logging.Level {
	if debug {
		return logging.DEBUG
	}
	return logging.INFO
}

func SetupLogger(levelFlag bool) {
	Backend := logging.NewLogBackend(os.Stderr, "", 0)
	formatedBackend := logging.NewBackendFormatter(Backend, format)

	BackendLevel := logging.AddModuleLevel(formatedBackend)
	BackendLevel.SetLevel(setupLevel(levelFlag), "")

	logging.SetBackend(BackendLevel)
}
