// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/bullyork/serviceGen/genCode"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate front-end code",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Work your own magic here
		if input == "" {
			fmt.Println("-i input yaml file must be specified")
			return
		}
		f, err1 := filepath.Abs(input)
		if err1 != nil {
			log.Fatalf("failed to get absoulte path of input idl file: %s", err1.Error())
		}
		viper.SetConfigType("yaml") // or viper.SetConfigType("YAML")
		viper.SetConfigFile(f)
		err2 := viper.ReadInConfig() // Find and read the config file
		if err2 != nil {             // Handle errors reading the config file
			panic(fmt.Errorf("Fatal error config file: %s \n", err2))
		}
		genCode.GenCode()
	},
}

var input string

func init() {
	RootCmd.AddCommand(genCmd)
	genCmd.PersistentFlags().StringVarP(&input, "input", "i", "", "input file")
}
