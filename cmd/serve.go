/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"github.com/ameyer8/ygs/internal/fileserver"

	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		dotfiles, err := cmd.Flags().GetBool("dotfiles")
		if err != nil {
			log.Fatal(err)
		}
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			log.Fatal(err)
		}
		var path string
		if len(args) == 0 {
			path, err = cmd.Flags().GetString("path")
			if err != nil {
				log.Fatal(err)
			}
		} else {
			path = args[0]
		}
		fs := fileserver.Server{
			Port:     port,
			Path:     path,
			Dotfiles: dotfiles,
		}

		SetupCloseHandler(&fs)

		fs.Start()

	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	//serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	serveCmd.Flags().Bool("dotfiles", false, "Allow serving dotfiles")

	serveCmd.Flags().IntP("port", "p", 8000, "Port for HTTP server")
	serveCmd.Flags().String("path", ".", "Port for HTTP server")
}

func SetupCloseHandler(fs *fileserver.Server) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed, exiting...")
		fs.TurnDownServer()

		os.Exit(0)
	}()
}
