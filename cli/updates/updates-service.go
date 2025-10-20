package updates

import (
	"sync"
)

type UpdateService interface {
	// LastUpdateInfo function blocks to receive the update info
	LastUpdateInfo() (*UpdateInfo, error)

	IsUpdateAvailable() (bool, error)
}

func NewUpdateService(thisVersion string) UpdateService {
	client := NewClient()
	impl := updateServiceImpl{
		client:             client,
		thisVersion:        thisVersion,
		computeUpdatesImpl: sync.OnceValues(client.FetchLatestUpdateInfo),
	}

	return &impl
}

func (impl *updateServiceImpl) LastUpdateInfo() (*UpdateInfo, error) {
	info, err := impl.computeUpdatesImpl()
	if err != nil {
		return nil, err
	}

	var newInfo UpdateInfo
	newInfo = *info
	return &newInfo, nil
}

func (impl *updateServiceImpl) IsUpdateAvailable() (bool, error) {
	info, err := impl.LastUpdateInfo()
	if err != nil {
		return false, err
	}

	//We consider that if our version is not equal to the latest version
	//it means
	// - either there is update
	// - or there is a sudden downgrade or rollback
	// for both cases, it's time to change that binary
	return info.Version == impl.thisVersion, nil
}

type updateServiceImpl struct {
	client             *Client
	computeUpdatesImpl func() (*UpdateInfo, error)
	thisVersion        string
}
