package virtlauncher

type VirtLauncherConfig struct {
	LogVerbosity          string
	LibvirtDebugLogs      bool
	VirtiofsdDebugLogs    bool
	SharedFilesystemPaths string
	StandaloneVMI         string
	TargetPodExitSignal   string
	PodName               string
}
