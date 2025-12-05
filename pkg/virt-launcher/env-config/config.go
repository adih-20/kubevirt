package env_config

import "os"

type VirtLauncherConfig struct {
	LogVerbosity          string
	LibvirtDebugLogs      bool
	VirtiofsdDebugLogs    bool
	SharedFilesystemPaths string
	StandaloneVMI         string
	TargetPodExitSignal   string
	PodName               string
}

func ReadVirtLauncherConfig() *VirtLauncherConfig {
	config := &VirtLauncherConfig{}

	if verbosityStr, ok := os.LookupEnv("VIRT_LAUNCHER_LOG_VERBOSITY"); ok {
		config.LogVerbosity = verbosityStr
	}

	if debugLogsStr, ok := os.LookupEnv("LIBVIRT_DEBUG_LOGS"); ok && debugLogsStr == "1" {
		config.LibvirtDebugLogs = true
	}

	if debugLogsStr, ok := os.LookupEnv("VIRTIOFSD_DEBUG_LOGS"); ok && debugLogsStr == "1" {
		config.VirtiofsdDebugLogs = true
	}

	if pathsStr, ok := os.LookupEnv("SHARED_FILESYSTEM_PATHS"); ok {
		config.SharedFilesystemPaths = pathsStr
	}

	if vmiStr, ok := os.LookupEnv("STANDALONE_VMI"); ok {
		config.StandaloneVMI = vmiStr
	}

	if signalStr, ok := os.LookupEnv("VIRT_LAUNCHER_TARGET_POD_EXIT_SIGNAL"); ok {
		config.TargetPodExitSignal = signalStr
	}

	if podName, ok := os.LookupEnv("POD_NAME"); ok {
		config.PodName = podName
	}

	return config
}
