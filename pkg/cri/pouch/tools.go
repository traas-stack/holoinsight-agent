package pouch

const (
	LabelPouchType    = "io.kubernetes.pouch.type"
	LabelValueSandbox = "sandbox"
)

func IsSandbox(labels map[string]string) bool {
	return labels[LabelPouchType] == LabelValueSandbox
}
