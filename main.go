package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
)

type myEventHandler struct {
	canal.DummyEventHandler

	pos         *syncPos
	saveSampler atomic.Uint64
}

func (h *myEventHandler) OnPosSynced(hdr *replication.EventHeader, pos mysql.Position, set mysql.GTIDSet, force bool) (err error) {
	slog.Info("on pos sync'ed",
		slog.Any("pos", pos),
		slog.Any("set", set),
		slog.Any("force", force),
	)
	// sampler 1/10
	if h.saveSampler.Add(1)%10 != 0 {
		return
	}
	*h.pos = syncPos(pos)
	return h.pos.save()
}

func (h *myEventHandler) OnTableChanged(hdr *replication.EventHeader, schema string, table string) (err error) {
	slog.Info("on table changed",
		slog.String("schema", schema),
		slog.String("table", table),
	)
	return
}

func (h *myEventHandler) OnDDL(hdr *replication.EventHeader, nextPos mysql.Position, queryEvent *replication.QueryEvent) (err error) {
	slog.Info("on DDL", slog.Any("nextPos", nextPos), slog.Any("queryEvent", queryEvent))
	return
}

func (h *myEventHandler) OnRow(e *canal.RowsEvent) (err error) {
	slog.Info("on row",
		slog.String("action", e.Action),
		slog.Any("rows", e.Rows),
	)
	return
}

func (h *myEventHandler) String() string { return fmt.Sprintf("myEventHandler") }

type syncPos mysql.Position

const lastBinlogPosFilename = ".last_binlog_pos.json"

func (r *syncPos) load() error {
	f, err := os.OpenFile(lastBinlogPosFilename, os.O_RDWR|os.O_CREATE, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(r); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

func (r *syncPos) save() error {
	f, err := os.OpenFile(lastBinlogPosFilename, os.O_RDWR|os.O_CREATE, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(r); err != nil {
		return err
	}
	return nil
}

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	var spos syncPos

	if err := spos.load(); err != nil {
		panic(err)
	}
	defer func() {
		println("saving")
		if x := spos.save(); x != nil {
			panic(x)
		}
	}()

	// See https://github.com/go-mysql-org/go-mysql

	cfg := canal.NewDefaultConfig()
	cfg.ServerID = 1337
	cfg.Addr = "127.0.0.1:3306"
	cfg.User = "root"
	cfg.Password = "admin123"

	cfg.Flavor = "mariadb"
	cfg.Dump.ExecutionPath = "mysqldump"

	cfg.Dump.TableDB = "testing_db"
	cfg.Dump.Tables = []string{"ent_assets"}

	c, err := canal.NewCanal(cfg)
	if err != nil {
		panic(err)
	}
	defer func() {
		println("setback")
		spos = syncPos(c.SyncedPosition())
	}()
	c.SetEventHandler(&myEventHandler{pos: &spos})

	go c.RunFrom(mysql.Position(spos))

	<-sig
	println("exited")
}
