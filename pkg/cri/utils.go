package cri

func ShortContainerId(cid string) string {
	if len(cid) <= 12 {
		return cid
	}
	return cid[:12]
}
