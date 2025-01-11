package layout_api

type ResolveLocallyAvailableIdeNotFound struct{}

func (e *ResolveLocallyAvailableIdeNotFound) Error() string {
	return "IDE is not available locally"
}
