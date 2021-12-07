package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var rootCmd = &cobra.Command{
	Use:   "pd-buddy",
	Short: "PagerDuty tools",
}

func Execute() error {

	rootCmd.AddCommand(incidentCmd())

	return rootCmd.Execute()
}

type config struct {
	Authtoken string
}

func client() *pagerduty.Client {
	b, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".pd.yml"))
	if err != nil {
		panic(err)
	}

	var conf config
	err = yaml.Unmarshal(b, &conf)
	if err != nil {
		panic(err)
	}

	return pagerduty.NewClient(conf.Authtoken)
}
