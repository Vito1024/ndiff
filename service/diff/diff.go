package diff

import (
	"context"
	"encoding/csv"
	"fmt"
	"ndiff"
	"ndiff/internal/clickhouse"
	"os"
	"sort"
	"sync"
)

const tag = "diff"

const (
	notInNewFileName = "not_in_new.csv"
	notInOldFileName = "not_in_old.csv"
)

type Service struct {
	old *clickhouse.DB
	new *clickhouse.DB

	notInNew *csv.Writer
	notInOld *csv.Writer

	processedHeight uint64

	tracker ndiff.Tracker
}

func New(old *clickhouse.DB, new *clickhouse.DB, tracker ndiff.Tracker) *Service {
	svc := &Service{
		old:     old,
		new:     new,
		tracker: tracker,
	}

	err := os.MkdirAll(ndiff.DIFF_RESULT_FILE_LOCATION, 0755)
	if err != nil {
		svc.tracker.Fatal(tag, "failed to create diff result file directory", ndiff.ErrorTag(err))
	}
	if _, err := os.Stat(fmt.Sprintf("%s/%s", ndiff.DIFF_RESULT_FILE_LOCATION, notInNewFileName)); err != nil {
		if !os.IsNotExist(err) {
			svc.tracker.Fatal(tag, "failed to get diff result file `not_in_new.csv`", ndiff.ErrorTag(err))
		}
		svc.tracker.Info(tag, "diff result file `not_in_new.csv` not exist, will create it")
		notInNewFile, err := os.OpenFile(fmt.Sprintf("%s/%s", ndiff.DIFF_RESULT_FILE_LOCATION, notInNewFileName), os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			svc.tracker.Fatal(tag, "failed to create diff result file `not_in_new.csv`", ndiff.ErrorTag(err))
		}
		svc.notInNew = csv.NewWriter(notInNewFile)
		err = svc.notInNew.Write([]string{"txid", "nft_type", "height", "nft_idx"})
		if err != nil {
			svc.tracker.Fatal(tag, "failed to write header to diff result file `not_in_new.csv`", ndiff.ErrorTag(err))
		}
	} else {
		svc.tracker.Info(tag, "diff result file `not_in_new.csv` already exist, will append to it")
		notInNewFile, err := os.OpenFile(fmt.Sprintf("%s/%s", ndiff.DIFF_RESULT_FILE_LOCATION, notInNewFileName), os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			svc.tracker.Fatal(tag, "failed to open diff result file `not_in_new.csv`", ndiff.ErrorTag(err))
		}
		svc.notInNew = csv.NewWriter(notInNewFile)
	}
	if _, err := os.Stat(fmt.Sprintf("%s/%s", ndiff.DIFF_RESULT_FILE_LOCATION, notInOldFileName)); err != nil {
		if !os.IsNotExist(err) {
			svc.tracker.Fatal(tag, "failed to get diff result file `not_in_old.csv`", ndiff.ErrorTag(err))
		}
		svc.tracker.Info(tag, "diff result file `not_in_old.csv` not exist, will create it")
		notInOldFile, err := os.OpenFile(fmt.Sprintf("%s/%s", ndiff.DIFF_RESULT_FILE_LOCATION, notInOldFileName), os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			svc.tracker.Fatal(tag, "failed to create diff result file `not_in_old.csv`", ndiff.ErrorTag(err))
		}
		svc.notInOld = csv.NewWriter(notInOldFile)
		err = svc.notInOld.Write([]string{"txid", "nft_type", "height", "nft_idx"})
		if err != nil {
			svc.tracker.Fatal(tag, "failed to write header to diff result file `not_in_old.csv`", ndiff.ErrorTag(err))
		}
	} else {
		svc.tracker.Info(tag, "diff result file `not_in_old.csv` already exist, will append to it")
		notInOldFile, err := os.OpenFile(fmt.Sprintf("%s/%s", ndiff.DIFF_RESULT_FILE_LOCATION, notInOldFileName), os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			svc.tracker.Fatal(tag, "failed to open diff result file `not_in_old.csv`", ndiff.ErrorTag(err))
		}
		svc.notInOld = csv.NewWriter(notInOldFile)
	}

	return svc
}

func (svc *Service) Diff(ctx context.Context) {
	svc.tracker.Debug(tag, "diff started.")
	svc.saveDiffResult(svc.diff(svc.rangeHeight(ctx)))
}

func (svc *Service) rangeHeight(ctx context.Context) <-chan NFTsForDiff {
	out := make(chan NFTsForDiff)

	go func() {
		defer close(out)
		defer svc.tracker.Info(tag, "range height exited")
		for height := ndiff.START_HEIGHT; height < ndiff.END_HEIGHT; height += ndiff.STEP {
			select {
			case <-ctx.Done():
				svc.tracker.Info(tag, "range height exit due to context done", ndiff.NewTag("start_height", height))
				return
			default:
				nftsForDiff := svc.fetchNFTs(ctx, height)
				if len(nftsForDiff.Old) > 0 || len(nftsForDiff.New) > 0 {
					out <- nftsForDiff
				}
				svc.tracker.Info(tag, "processed height, no nfts", ndiff.NewTag("start_height", height), ndiff.NewTag("to_height", height+ndiff.STEP))
			}
		}
	}()

	return out
}

func (svc *Service) diff(in <-chan NFTsForDiff) <-chan DiffResult {
	out := make(chan DiffResult)

	go func() {
		defer close(out)
		for nftsForDiff := range in {

			var diffResult DiffResult
			diffResult.StartHeight = nftsForDiff.StartHeight
			diffResult.NotInNew = make([]NFT, 0)
			diffResult.NotInOld = make([]NFT, 0)

			for oldTxId, oldNFT := range nftsForDiff.Old {
				if _, ok := nftsForDiff.New[oldTxId]; !ok {
					diffResult.NotInNew = append(diffResult.NotInNew, oldNFT)
				}
			}
			for newTxId, newNFT := range nftsForDiff.New {
				if _, ok := nftsForDiff.Old[newTxId]; !ok {
					diffResult.NotInOld = append(diffResult.NotInOld, newNFT)
				}
			}

			// sort by height and nftidx, asc
			sort.Slice(diffResult.NotInNew, func(i, j int) bool {
				if diffResult.NotInNew[i].Height != diffResult.NotInNew[j].Height {
					return diffResult.NotInNew[i].Height < diffResult.NotInNew[j].Height
				}
				return diffResult.NotInNew[i].NFTIdx < diffResult.NotInNew[j].NFTIdx
			})
			sort.Slice(diffResult.NotInOld, func(i, j int) bool {
				if diffResult.NotInOld[i].Height != diffResult.NotInOld[j].Height {
					return diffResult.NotInOld[i].Height < diffResult.NotInOld[j].Height
				}
				return diffResult.NotInOld[i].NFTIdx < diffResult.NotInOld[j].NFTIdx
			})

			if len(diffResult.NotInNew) == 0 && len(diffResult.NotInOld) == 0 {
				svc.tracker.Info(tag, "processed height, no diff", ndiff.NewTag("start_height", diffResult.StartHeight), ndiff.NewTag("to_height", diffResult.StartHeight+ndiff.STEP))
				continue
			} else {
				svc.tracker.Info(tag, "diff: ", ndiff.NewTag("not_in_new", len(diffResult.NotInNew)), ndiff.NewTag("not_in_old", len(diffResult.NotInOld)))
			}

			out <- diffResult
		}
	}()

	return out
}

func (svc *Service) saveDiffResult(in <-chan DiffResult) {
	defer svc.notInNew.Flush()
	defer svc.notInOld.Flush()

	for diffResult := range in {
		for _, nft := range diffResult.NotInNew {
			err := svc.notInNew.Write([]string{nft.TxId, fmt.Sprintf("%d", nft.NFTType), fmt.Sprintf("%d", nft.Height), fmt.Sprintf("%d", nft.NFTIdx)})
			if err != nil {
				svc.tracker.Fatal(tag, "failed to write diff result to `not_in_new.csv`", ndiff.ErrorTag(err), ndiff.NewTag("nft", nft))
			}
		}
		for _, nft := range diffResult.NotInOld {
			err := svc.notInOld.Write([]string{nft.TxId, fmt.Sprintf("%d", nft.NFTType), fmt.Sprintf("%d", nft.Height), fmt.Sprintf("%d", nft.NFTIdx)})
			if err != nil {
				svc.tracker.Fatal(tag, "failed to write diff result to `not_in_old.csv`", ndiff.ErrorTag(err), ndiff.NewTag("nft", nft))
			}
		}
		svc.notInNew.Flush()
		if svc.notInNew.Error() != nil {
			svc.tracker.Fatal(tag, "failed to flush diff result to `not_in_new.csv`", ndiff.ErrorTag(svc.notInNew.Error()))
		}
		svc.notInOld.Flush()
		if svc.notInOld.Error() != nil {
			svc.tracker.Fatal(tag, "failed to flush diff result to `not_in_old.csv`", ndiff.ErrorTag(svc.notInOld.Error()))
		}

		svc.processedHeight = diffResult.StartHeight + ndiff.STEP - 1
		svc.tracker.Info(tag, "processed height, diff result has been saved to file", ndiff.NewTag("start_height", diffResult.StartHeight), ndiff.NewTag("to_height", diffResult.StartHeight+ndiff.STEP))
	}
}

func (svc *Service) fetchNFTs(ctx context.Context, startHeight uint64) NFTsForDiff {
	queryFn := func(db *clickhouse.DB, sql string) (map[string]NFT, error) {
		rows, err := db.QueryAll(ctx, sql, startHeight, startHeight+ndiff.STEP)
		if err == clickhouse.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			svc.tracker.Fatal(tag, "failed to fetch nfts from clickhouse", ndiff.ErrorTag(err), ndiff.NewTag("start_height", startHeight), ndiff.NewTag("db", db.Name()))
			return nil, err
		}
		nfts := make(map[string]NFT, 72000)
		for rows.Next() {
			var nft NFT
			if err := rows.Scan(&nft.TxId, &nft.Height, &nft.NFTIdx, &nft.NFTType); err != nil {
				svc.tracker.Fatal(tag, "failed to scan nft", ndiff.ErrorTag(err), ndiff.NewTag("start_height", startHeight), ndiff.NewTag("db", db.Name()))
				return nil, err
			}
			nfts[nft.TxId] = nft
		}
		return nfts, nil
	}

	var diffNFT NFTsForDiff
	diffNFT.StartHeight = startHeight

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		oldNFTs, err := queryFn(svc.old, sqlOldSelectNFTByHeightRange)
		if err != nil {
			svc.tracker.Fatal(tag, "failed to fetch nfts from old db", ndiff.ErrorTag(err), ndiff.NewTag("start_height", startHeight), ndiff.NewTag("db", svc.old.Name()))
			return
		}
		diffNFT.Old = oldNFTs
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		newNFTs, err := queryFn(svc.new, sqlNewSelectNFTByHeightRange)
		if err != nil {
			svc.tracker.Fatal(tag, "failed to fetch nfts from new db", ndiff.ErrorTag(err), ndiff.NewTag("start_height", startHeight), ndiff.NewTag("db", svc.new.Name()))
			return
		}
		diffNFT.New = newNFTs
	}()
	wg.Wait()
	svc.tracker.Debug(tag, "fetched nfts", ndiff.NewTag("start_height", startHeight), ndiff.NewTag("old", len(diffNFT.Old)), ndiff.NewTag("new", len(diffNFT.New)))

	return diffNFT
}
