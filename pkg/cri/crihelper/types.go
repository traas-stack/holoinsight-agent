package crihelper

type (
	ProcessInfo struct {
		User         string   `json:"user"`
		Name         string   `json:"name"`
		CmdlineSlice []string `json:"cmdlineSlice"`
	}
)
