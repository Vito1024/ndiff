package main

import (
	"context"
	"flag"
	"ndiff"
	"ndiff/config"
	"ndiff/internal/clickhouse"
	"ndiff/service/diff"
	"ndiff/tracker"
	"os"
	"os/signal"
	"syscall"
)

type dep struct {
	tracker ndiff.Tracker
	config  config.Config

	// internal
	old *clickhouse.DB
	new *clickhouse.DB
}

func main() {
	configPath := flag.String("config", "", "config file path")
	flag.Parse()

	var dep dep
	dep.initInfra(*configPath)
	dep.initStorage()

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	go func() {
		s := <-sig
		dep.tracker.Info(ndiff.SERVICE_NAME, "received signal", ndiff.NewTag("signal", s))
		cancel()
	}()

	diffSvc := diff.New(dep.old, dep.new, dep.tracker)
	diffSvc.Diff(ctx)

	dep.tracker.Info(ndiff.SERVICE_NAME, "exited")
}

func (d *dep) initInfra(configFilepath string) {
	d.config = config.New(configFilepath)
	d.tracker = tracker.New()

	d.tracker.Info(ndiff.SERVICE_NAME, "successfully loaded config", ndiff.NewTag("config", d.config))
}

func (d *dep) initStorage() {
	d.old = clickhouse.New("old", d.config.Old, d.tracker)
	d.new = clickhouse.New("new", d.config.New, d.tracker)

	d.tracker.Info(ndiff.SERVICE_NAME, "successfully initialized storage")
}
