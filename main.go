package main

import (
	"fmt"
	"log"

	"github.com/chriserin/sq/internal/config"
	"github.com/chriserin/sq/internal/mappings"
	"github.com/chriserin/sq/internal/seqmidi"
	"github.com/chriserin/sq/internal/themes"
	"github.com/spf13/cobra"
)

const VERSION = "v0.1.0-beta.3"

type ProgramOptions struct {
	gridTemplate string
	instrument   string
	outport      bool
	theme        string
	midiout      string
}

var cliOptions ProgramOptions

func main() {
	rootCmd := &cobra.Command{
		Use:   "sq",
		Short: "A sequencer for your cli",
		Long:  "A sequencer for your cli",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"sq"}, cobra.ShellCompDirectiveFilterFileExt
		},
		Run: func(cmd *cobra.Command, args []string) {

			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Recovered from panic: %v\n", r)
				}
			}()

			var err error
			var filename string
			if len(args) > 0 {
				filename = args[0]
			}
			p, err := RunProgram(filename, cliOptions)
			_, err = p.Run()
			if err != nil {
				log.Fatalf("Program Failure: %v\n", err)
			} else {
				return
			}
		},
	}

	cmdVersion := &cobra.Command{
		Use:   "version",
		Short: "Version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("sq v%s\n", VERSION)
		},
	}

	cmdListOutports := &cobra.Command{
		Use:   "list",
		Short: "List available midi outports",
		Run: func(cmd *cobra.Command, args []string) {
			outports, _ := seqmidi.Outs()
			for i, outport := range outports {
				fmt.Printf("%d) %s\n", i+1, outport)
			}
		},
	}

	cmdMappings := &cobra.Command{
		Use:   "mappings",
		Short: "List all keyboard mappings with descriptions",
		Run: func(cmd *cobra.Command, args []string) {
			allMappings := mappings.GetAllMappings()
			for _, m := range allMappings {
				fmt.Printf("%-20s %-30s %s\n", m.GetKeys(), m.Name, m.Description)
			}
		},
	}

	rootCmd.AddCommand(cmdListOutports)
	rootCmd.AddCommand(cmdVersion)
	rootCmd.AddCommand(cmdMappings)
	rootCmd.Flags().StringVar(&cliOptions.gridTemplate, "template", "Drums", "Choose a template (default: Drums)")
	rootCmd.Flags().StringVar(&cliOptions.instrument, "instrument", "Standard", "Choose an instrument for CC integration (default: Standard)")
	rootCmd.Flags().BoolVar(&cliOptions.outport, "outport", false, "sq will create an outport to send midi")
	rootCmd.Flags().StringVar(&cliOptions.theme, "theme", "miles", "Choose an theme for the sequencer visual representation")
	rootCmd.Flags().StringVar(&cliOptions.midiout, "midiout", "", "Choose a midi out port")

	// Register completion function for template flag
	err := rootCmd.RegisterFlagCompletionFunc("template", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		config.Init()
		return config.GetTemplateNames(), cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		log.Fatal("Failed to register template completion")
	}

	// Register completion function for template flag
	err = rootCmd.RegisterFlagCompletionFunc("instrument", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		config.Init()
		return config.GetInstrumentNames(), cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		log.Fatal("Failed to register template completion")
	}

	// Register completion function for theme flag
	err = rootCmd.RegisterFlagCompletionFunc("theme", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return themes.Themes, cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		log.Fatal("Failed to register theme completion")
	}

	// Register completion function for midiout flag
	err = rootCmd.RegisterFlagCompletionFunc("midiout", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		outports, _ := seqmidi.Outs()
		outportNames := make([]string, len(outports))
		for i, outport := range outports {
			outportNames[i] = fmt.Sprintf("%s", outport)
		}
		return outportNames, cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		log.Fatal("Failed to register midiout completion")
	}

	err = rootCmd.Execute()
	if err != nil {
		log.Fatal("Program failed")
	}
}
