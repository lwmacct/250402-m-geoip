package start

import (
	"github.com/lwmacct/250402-m-geoip/api"
	"github.com/lwmacct/250402-m-geoip/app"

	"github.com/lwmacct/250300-go-mod-mflag/pkg/mflag"
	"github.com/lwmacct/250300-go-mod-mlog/pkg/mlog"
	"github.com/spf13/cobra"
)

func Cmd() *mflag.Ts {
	mc := mflag.New(app.Flag).UsePackageName("")
	mc.AddCmd(func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	}, "run", "", "app", "mlog")

	mc.AddCmd(func(cmd *cobra.Command, args []string) {
		test(cmd, args)
	}, "test", "", "app", "mlog")

	return mc
}

func run(cmd *cobra.Command, args []string) {
	_ = map[string]any{"cmd": cmd, "args": args}
	mlog.Info(mlog.H{"msg": "app.Flag", "data": app.Flag})
	api.New().InitDb(app.Flag.App.DSN.PGSQL).Run()
	mlog.Close()

}

func test(cmd *cobra.Command, args []string) {
	_ = map[string]any{"cmd": cmd, "args": args}
	defer mlog.Close()
	mlog.Info(mlog.H{"msg": "test"})

	select {}

}
