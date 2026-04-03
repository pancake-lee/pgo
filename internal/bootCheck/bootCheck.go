package main

import (
	"flag"

	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/pkg/papp"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pmq"
	"github.com/pancake-lee/pgo/pkg/predis"
)

func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	pconfig.MustInitConfig(*c)
	plogger.InitFromConfig(*l)
	plogger.Info("BootCheck started...")

	if !pconfig.Has(predis.DefaultConfigGroup) {
		plogger.Info("Redis config not found, skipping check.")
	} else {
		if err := papp.CheckRedis(); err != nil {
			plogger.Fatalf("Redis check failed: %v", err)
		}
	}

	if !pconfig.Has(pmq.DefaultConfigGroup) {
		plogger.Info("RabbitMQ config not found, skipping check.")
	} else {
		if err := papp.CheckRabbitMQ(); err != nil {
			plogger.Fatalf("RabbitMQ check failed: %v", err)
		}
	}

	// Includes: Create DB if not exists, AutoMigrate
	if !pconfig.Has(pdb.DefaultConfigGroup) {
		plogger.Info("Mysql config not found, skipping check.")
	} else {
		err := papp.CheckMysql([]any{
			&model.AbandonCode{},
			&model.CourseSwapRequest{},
			&model.Project{},
			&model.Task{},
			&model.User{},
			&model.UserDept{},
			&model.UserDeptAssoc{},
			&model.UserJob{},
			&model.UserProjectAssoc{},
			&model.UserRole{},
			&model.UserRoleAssoc{},
			&model.UserRolePermissionAssoc{},
		})
		if err != nil {
			plogger.Fatalf("MySQL check failed: %v", err)
		}
	}

	plogger.Info("BootCheck finished successfully.")
}
