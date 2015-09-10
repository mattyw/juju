// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgrades

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/juju/errors"
	"github.com/juju/utils"
	"github.com/juju/utils/symlink"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/state/multiwatcher"
)

var (
	rootLogDir   = "/var/log"
	rootSpoolDir = "/var/spool/rsyslog"
)

var chownPath = utils.ChownPath

var isLocalEnviron = func(envConfig *config.Config) bool {
	return envConfig.Type() == "local"
}

func migrateLocalProviderAgentConfig(context Context) error {
	st := context.State()
	if st == nil {
		logger.Debugf("no state connection, no migration required")
		// We're running on a different node than the state server.
		return nil
	}
	envConfig, err := st.EnvironConfig()
	if err != nil {
		return fmt.Errorf("failed to read current config: %v", err)
	}
	if !isLocalEnviron(envConfig) {
		logger.Debugf("not a local environment, no migration required")
		return nil
	}
	attrs := envConfig.AllAttrs()
	rootDir, _ := attrs["root-dir"].(string)
	sharedStorageDir := filepath.Join(rootDir, "shared-storage")
	// In case these two are empty we need to set them and update the
	// environment config.
	namespace, _ := attrs["namespace"].(string)
	container, _ := attrs["container"].(string)

	if namespace == "" {
		username := os.Getenv("USER")
		if username == "root" {
			// sudo was probably called, get the original user.
			username = os.Getenv("SUDO_USER")
		}
		if username == "" {
			return fmt.Errorf("cannot get current user from the environment: %v", os.Environ())
		}
		namespace = username + "-" + envConfig.Name()
	}
	if container == "" {
		container = "lxc"
	}

	dataDir := rootDir
	localLogDir := filepath.Join(rootDir, "log")
	metricSpoolDir := filepath.Join(rootDir, "metricspool")
	uniterStateDir := filepath.Join(rootDir, "uniter", "state")
	// rsyslogd is restricted to write to /var/log
	logDir := fmt.Sprintf("%s/juju-%s", rootLogDir, namespace)
	jobs := []multiwatcher.MachineJob{multiwatcher.JobManageEnviron}
	values := map[string]string{
		agent.Namespace: namespace,
		// ContainerType is empty on the bootstrap node.
		agent.ContainerType:    "",
		agent.AgentServiceName: "juju-agent-" + namespace,
	}
	deprecatedValues := []string{
		"SHARED_STORAGE_ADDR",
		"SHARED_STORAGE_DIR",
	}

	// Remove shared-storage dir if there.
	if err := os.RemoveAll(sharedStorageDir); err != nil {
		return fmt.Errorf("cannot remove deprecated %q: %v", sharedStorageDir, err)
	}

	// We need to create the dirs if they don't exist.
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("cannot create dataDir %q: %v", dataDir, err)
	}
	// We always recreate the logDir to make sure it's empty.
	if err := os.RemoveAll(logDir); err != nil {
		return fmt.Errorf("cannot remove logDir %q: %v", logDir, err)
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("cannot create logDir %q: %v", logDir, err)
	}
	if err := os.MkdirAll(metricSpoolDir, 0755); err != nil {
		return fmt.Errorf("cannot create metricSpoolDir %q: %v", metricSpoolDir, err)
	}
	if err := os.MkdirAll(uniterStateDir, 0755); err != nil {
		return fmt.Errorf("cannot create uniterStateDir %q: %v", uniterStateDir, err)
	}
	// Reconfigure rsyslog as needed:
	// 1. logDir must be owned by syslog:adm
	// 2. Remove old rsyslog spool config
	// 3. Relink logs to the new logDir
	if err := chownPath(logDir, "syslog"); err != nil {
		return err
	}
	spoolConfig := fmt.Sprintf("%s/machine-0-%s", rootSpoolDir, namespace)
	if err := os.RemoveAll(spoolConfig); err != nil {
		return fmt.Errorf("cannot remove %q: %v", spoolConfig, err)
	}
	allMachinesLog := filepath.Join(logDir, "all-machines.log")
	if err := symlink.New(allMachinesLog, localLogDir+"/"); err != nil && !os.IsExist(err) {
		return fmt.Errorf("cannot symlink %q to %q: %v", allMachinesLog, localLogDir, err)
	}
	machine0Log := filepath.Join(localLogDir, "machine-0.log")
	if err := symlink.New(machine0Log, logDir+"/"); err != nil && !os.IsExist(err) {
		return fmt.Errorf("cannot symlink %q to %q: %v", machine0Log, logDir, err)
	}

	newCfg := map[string]interface{}{
		"namespace": namespace,
		"container": container,
	}
	if err := st.UpdateEnvironConfig(newCfg, nil, nil); err != nil {
		return fmt.Errorf("cannot update environment config: %v", err)
	}

	return context.AgentConfig().Migrate(agent.MigrateParams{
		Paths: agent.Paths{
			DataDir:         dataDir,
			LogDir:          logDir,
			MetricsSpoolDir: metricSpoolDir,
		},
		Jobs:         jobs,
		Values:       values,
		DeleteValues: deprecatedValues,
	})
}

func addEnvironmentUUIDToAgentConfig(context Context) error {
	if context.AgentConfig().Environment().Id() != "" {
		logger.Infof("environment uuid already set in agent config")
		return nil
	}

	environTag, err := context.APIState().EnvironTag()
	if err != nil {
		return errors.Annotate(err, "no environment uuid set on api")
	}

	return context.AgentConfig().Migrate(agent.MigrateParams{
		Environment: environTag,
	})
}
