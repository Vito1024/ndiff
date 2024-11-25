package diff

const sqlOldSelectNFTByHeightRange = `
SELECT
	lower(hex(reverse(txid))) as txid, height, nftidx, nfttype
FROM
	blknft_height
WHERE
	height >= ? AND height < ?
	AND nfttype in (3, 5)
`

const sqlNewSelectNFTByHeightRange = `
SELECT
	lower(hex(reverse(txid))) as txid, height, nftidx, nfttype
FROM
	blknft_height
WHERE
	height >= ? AND height < ?
	AND nfttype = 3
`
