/*
Copyright 2017 The Nuclio Authors.

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

package runtime

import (
	"fmt"
	"path"
	"strings"

	"github.com/nuclio/nuclio/pkg/cmdrunner"
	"github.com/nuclio/nuclio/pkg/common"
	"github.com/nuclio/nuclio/pkg/dockerclient"
	"github.com/nuclio/nuclio/pkg/errors"
	"github.com/nuclio/nuclio/pkg/functionconfig"
	"github.com/nuclio/nuclio/pkg/version"

	"github.com/nuclio/logger"
)

type ProcessorDockerfileInfo struct {
	BaseImage            string
	OnbuildImage         string
	OnbuildArtifactPaths map[string]string
	ImageArtifactPaths   map[string]string
	Directives           []string
}

type Runtime interface {

	// DetectFunctionHandlers returns a list of all the handlers
	// in that directory given a path holding a function (or functions)
	DetectFunctionHandlers(functionPath string) ([]string, error)

	// OnAfterStagingDirCreated prepares anything it may need in that directory
	// towards building a functioning processor,
	OnAfterStagingDirCreated(stagingDir string) error

	// GetProcessorDockerfileInfo returns information required to build the processor Dockerfile
	GetProcessorDockerfileInfo(versionInfo *version.Info) (*ProcessorDockerfileInfo, error)

	// GetName returns the name of the runtime, including version if applicable
	GetName() string

	// GetProcessorImageObjectPaths returns the paths of all objects that should reside in the handler
	// directory
	GetHandlerDirObjectPaths() []string
}

type Factory interface {
	Create(logger.Logger, string, *functionconfig.Config) (Runtime, error)
}

type AbstractRuntime struct {
	Logger         logger.Logger
	StagingDir     string
	FunctionConfig *functionconfig.Config
	DockerClient   dockerclient.Client
	CmdRunner      cmdrunner.CmdRunner
}

func NewAbstractRuntime(logger logger.Logger,
	stagingDir string,
	functionConfig *functionconfig.Config) (*AbstractRuntime, error) {
	var err error

	newRuntime := &AbstractRuntime{
		Logger:         logger,
		StagingDir:     stagingDir,
		FunctionConfig: functionConfig,
	}

	newRuntime.CmdRunner, err = cmdrunner.NewShellRunner(newRuntime.Logger)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create command runner")
	}

	// create a docker client
	newRuntime.DockerClient, err = dockerclient.NewShellClient(newRuntime.Logger, newRuntime.CmdRunner)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create docker client")
	}

	return newRuntime, nil
}

func (ar *AbstractRuntime) OnAfterStagingDirCreated(stagingDir string) error {
	return nil
}

// return a map of objects the runtime needs to copy into the processor image
// the key can be a dir, a file or a url of a file
// the value is an absolute path into the docker image
func (ar *AbstractRuntime) GetProcessorImageObjectPaths() map[string]string {
	return nil
}

func (ar *AbstractRuntime) GetFunctionDir() string {

	// if the function directory was passed, just return that. if the function path was passed, return the directory
	// the function is in
	if common.IsDir(ar.FunctionConfig.Spec.Build.Path) {
		return ar.FunctionConfig.Spec.Build.Path
	}

	return path.Dir(ar.FunctionConfig.Spec.Build.Path)
}

// GetRuntimeNameAndVersion returns name and version of runtime from runtime.
// e.g. go:1.8 -> go, 1.8
func (ar *AbstractRuntime) GetRuntimeNameAndVersion() (string, string) {
	nameAndVersion := strings.Split(ar.FunctionConfig.Spec.Runtime, ":")

	switch len(nameAndVersion) {

	// if both are passed (e.g. python:3.6) - return them both
	case 2:
		return nameAndVersion[0], nameAndVersion[1]

	// otherwise - return the first element (e.g. go -> go)
	default:
		return nameAndVersion[0], ""
	}
}

// GetProcessorImageObjectPaths returns the paths of all objects that should reside in the handler
// directory
func (ar *AbstractRuntime) GetHandlerDirObjectPaths() []string {

	// by default, just return the build path
	return []string{ar.FunctionConfig.Spec.Build.Path}
}

// DetectFunctionHandlers returns a list of all the handlers
// in that directory given a path holding a function (or functions)
func (ar *AbstractRuntime) DetectFunctionHandlers(functionPath string) ([]string, error) {

	// use the function path: /some/path/func.py -> func
	functionFileName := path.Base(ar.FunctionConfig.Spec.Build.Path)
	functionFileName = functionFileName[:len(functionFileName)-len(path.Ext(functionFileName))]

	return []string{fmt.Sprintf("%s:%s", functionFileName, "handler")}, nil
}
