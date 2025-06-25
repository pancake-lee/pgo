package psql

import (
	"os"

	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/spf13/cobra"
)

// 在makefile中使用了psql命令，但是在不同系统环境下，遇到了一些问题，干脆自己写一个简单的替代

var PsqlCmd = &cobra.Command{
	Use:   "psql",
	Short: "Simple PostgreSQL client",
	Long:  "A simple PostgreSQL client to replace psql command in Makefile",
	Run:   run,
}

var (
	host     string
	port     int32
	user     string
	password string = os.Getenv("PGPASSWORD")
	database string
	command  string
	filename string
)

func init() {
	// 全局参数
	PsqlCmd.Flags().StringVar(&host, "host", "localhost", "PostgreSQL host") // -h会和help冲突
	PsqlCmd.Flags().Int32VarP(&port, "port", "p", 5432, "PostgreSQL port")
	PsqlCmd.Flags().StringVarP(&user, "user", "U", "root", "PostgreSQL user")
	PsqlCmd.Flags().StringVarP(&database, "database", "d", "postgres", "PostgreSQL database")

	PsqlCmd.Flags().StringVarP(&filename, "filename", "f", "", "SQL file to execute")
	PsqlCmd.Flags().StringVarP(&command, "command", "c", "", "SQL command to execute")
}

func run(cmd *cobra.Command, args []string) {
	err := pdb.InitPG(host, user, password, database, port)
	if err != nil {
		plogger.Errorf("Error: %v\n", err)
		os.Exit(1)
	}

	if command != "" {
		plogger.Debugf("sql cmd  [%s]", command)
		_, err = pdb.Exec(command)
		if err != nil {
			plogger.Errorf("Error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if filename != "" {
		plogger.Debugf("sql file[%s]", filename)
		err = pdb.ExecFile(filename)
		if err != nil {
			plogger.Errorf("Error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	plogger.Errorf("Error: either command or file must be provided")
}
