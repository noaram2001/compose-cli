/*
   Copyright 2020 Docker Compose CLI authors

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

package compose

import (
	"context"

	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/docker/compose-cli/api/client"
	"github.com/docker/compose-cli/context/store"
	"github.com/docker/compose-cli/errdefs"
)

type composeOptions struct {
	Name        string
	DomainName  string
	WorkingDir  string
	ConfigPaths []string
	Environment []string
	Format      string
	Detach      bool
	Quiet       bool
}

func addComposeCommonFlags(f *pflag.FlagSet, opts *composeOptions) {
	f.StringVarP(&opts.Name, "project-name", "p", "", "Project name")
	f.StringVar(&opts.Format, "format", "", "Format the output. Values: [pretty | json]. (Default: pretty)")
	f.BoolVarP(&opts.Quiet, "quiet", "q", false, "Only display IDs")
}

func (o *composeOptions) toProjectName() (string, error) {
	if o.Name != "" {
		return o.Name, nil
	}

	options, err := o.toProjectOptions()
	if err != nil {
		return "", err
	}

	project, err := cli.ProjectFromOptions(options)
	if err != nil {
		return "", err
	}
	return project.Name, nil
}

func (o *composeOptions) toProjectOptions() (*cli.ProjectOptions, error) {
	return cli.NewProjectOptions(o.ConfigPaths,
		cli.WithOsEnv,
		cli.WithEnv(o.Environment),
		cli.WithWorkingDirectory(o.WorkingDir),
		cli.WithName(o.Name))
}

// Command returns the compose command with its child commands
func Command(contextType string) *cobra.Command {
	command := &cobra.Command{
		Short: "Docker Compose",
		Use:   "compose",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return checkComposeSupport(cmd.Context())
		},
	}

	command.AddCommand(
		upCommand(contextType),
		downCommand(),
		psCommand(),
		listCommand(),
		logsCommand(),
		convertCommand(),
	)

	if contextType == store.LocalContextType {
		command.AddCommand(
			buildCommand(),
			pushCommand(),
			pullCommand(),
		)
	}

	return command
}

func checkComposeSupport(ctx context.Context) error {
	_, err := client.New(ctx)
	if errdefs.IsNotFoundError(err) {
		return errdefs.ErrNotImplemented
	}

	return err
}

//
func filter(project *types.Project, services []string) error {
	if len(services) == 0 {
		// All services
		return nil
	}

	names := map[string]bool{}
	err := addServiceNames(project, services, names)
	if err != nil {
		return err
	}
	var filtered types.Services
	for _, s := range project.Services {
		if _, ok := names[s.Name]; ok {
			filtered = append(filtered, s)
		}
	}
	project.Services = filtered
	return nil
}

func addServiceNames(project *types.Project, services []string, names map[string]bool) error {
	for _, name := range services {
		names[name] = true
		service, err := project.GetService(name)
		if err != nil {
			return err
		}
		err = addServiceNames(project, service.GetDependencies(), names)
		if err != nil {
			return err
		}
	}
	return nil
}
