package types

type RemoteConfigStorage interface {
	Read() (RemoteConfigData, error)
	Write(RemoteConfigData) error
}
