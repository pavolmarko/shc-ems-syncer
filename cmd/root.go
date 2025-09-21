package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/pavolmarko/shc-ems-syncer/config"
	"github.com/pavolmarko/shc-ems-syncer/ems"
	"github.com/pavolmarko/shc-ems-syncer/shc"
)

var (
	// Used for flags.
	configPath string
)

var rootCmd = &cobra.Command{
	Use:          "shc-ems-syncer",
	Short:        "shc-ems-syncer is tool to sync Bosch SmartHomeController (SHC) data to an EMS device",
	Long:         `See https://github.com/pavolmarko/shc-ems-syncer`,
	SilenceUsage: true, // https://github.com/spf13/cobra/issues/340
	// No Run() or RunE() - reqruies subcommand
}

var emsCmd = &cobra.Command{
	Use:   "ems",
	Short: "interact with EMS-ESP device",
	// No Run() or RunE() - reqruies subcommand
}

var emsPingCmd = &cobra.Command{
	Use:   "ping",
	Short: "tests connection to EMS-ESP",
	RunE: func(cmd *cobra.Command, args []string) error {
		return emsPing()
	},
}

var shcCmd = &cobra.Command{
	Use:   "shc",
	Short: "interact with SHC device",
	// No Run() or RunE() - reqruies subcommand
}

var shcPingCmd = &cobra.Command{
	Use:   "ping",
	Short: "tests connection to SHC",
	RunE: func(cmd *cobra.Command, args []string) error {
		return shcPing()
	},
}

var shcRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "registers this client with SHC",
	Long: `register uses the https://{{schhost}}:8443/smarthome/clients endpoint
  to register this client (identified by its client certificate, as given in the config)
  with the Bosch Smart Home Controller (SHC).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return shcRegister()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file")
	rootCmd.MarkPersistentFlagRequired("config")

	rootCmd.AddCommand(emsCmd)
	emsCmd.AddCommand(emsPingCmd)

	rootCmd.AddCommand(shcCmd)
	shcCmd.AddCommand(shcPingCmd)
	shcCmd.AddCommand(shcRegisterCmd)
}

func emsPing() error {
	cfg, err := config.Read(configPath)
	if err != nil {
		return fmt.Errorf("can't read config: %v", err)
	}

	emsCli := ems.CreateClient(cfg.EmsEspHostport, cfg.EmsEspAccessToken)
	res, err := emsCli.Ping()
	if err != nil {
		return fmt.Errorf("can't ping ems: %v", err)
	}

	fmt.Println(res)

	return nil
}

func shcPing() error {
	cfg, err := config.Read(configPath)
	if err != nil {
		return fmt.Errorf("can't read config: %v", err)
	}

	shcCli := shc.CreateClient(cfg.ShcHost, cfg.ShcCaCertPool, cfg.ShcClientCert)

	res, err := shcCli.Ping()
	if err != nil {
		return fmt.Errorf("can't ping shc: %v", err)
	}
	fmt.Println(res)

	return nil
}

func shcRegister() error {
	cfg, err := config.Read(configPath)
	if err != nil {
		return fmt.Errorf("can't read config: %v", err)
	}

	shcCli := shc.CreateClient(cfg.ShcHost, cfg.ShcCaCertPool, cfg.ShcClientCert)

	fmt.Println("Enter Bosch Smart Home system password:")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("can't read password: from stdin: %v", err)
	}

	systemPassword := string(bytePassword)

	res, err := shcCli.Register(systemPassword)
	if err != nil {
		return fmt.Errorf("register failed: %v", err)
	}

	fmt.Println("Registration successful.")
	if res != "" {
		fmt.Println("Message from SHC:")
		fmt.Println(res)
	}

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
