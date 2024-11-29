package diff

type NFT struct {
	TxId      string
	Height    uint32
	NFTIdx    uint64
	NFTType   uint8
	NFTNumber int64
}

type NFTsForDiff struct {
	StartHeight uint64
	Old         map[string]NFT // txId -> NFT
	New         map[string]NFT // txId -> NFT
}

type DiffResult struct {
	StartHeight uint64
	NotInNew    []NFT // old - new
	NotInOld    []NFT // new - old
}
