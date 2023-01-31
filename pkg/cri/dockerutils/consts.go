package dockerutils

const (
	MergedDir            = "MergedDir"
	LabelDockerType      = "io.kubernetes.docker.type"
	LabelValuePodSandbox = "podsandbox"
)

func IsSandbox(labels map[string]string) bool {
	return labels[LabelDockerType] == LabelValuePodSandbox
}
