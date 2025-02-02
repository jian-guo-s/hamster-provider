package config

import (
	"github.com/hamster-shared/hamster-provider/core/modules/utils"
)

// VmOption vm configin formation
type VmOption struct {
	Cpu        uint64
	Mem        uint64
	Disk       uint64
	System     string
	Image      string
	AccessPort int
	// virtualization type,docker/kvm
	Type string
}

// ConfigVM  config
func (cm *ConfigManager) ConfigVM(vmOption VmOption) error {

	config, err := cm.GetConfig()
	if err != nil {
		return err
	}
	config.Vm = vmOption

	err = cm.Save(config)
	return err
}

func (cm *ConfigManager) AddBootstrap(bootstrap string) error {
	config, err := cm.GetConfig()
	if err != nil {
		return err
	}

	bootstraps := config.Bootstraps

	if utils.Contains(bootstraps, bootstrap) {
		return nil
	} else {
		config.Bootstraps = append(bootstraps, bootstrap)
	}

	return cm.Save(config)
}

func (cm *ConfigManager) RemoveBootstrap(bootstrap string) error {
	config, err := cm.GetConfig()
	if err != nil {
		return err
	}

	config.Bootstraps = utils.Remove(config.Bootstraps, bootstrap)

	return cm.Save(config)
}
