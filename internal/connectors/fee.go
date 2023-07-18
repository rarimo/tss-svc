package connectors

// That file contains the methods to approximate and calculate Cosmos transaction gas price and fee.
// Currently, our validator nodes requires 0urmo minimal fee so the `GetFeeAmount` always returns 0.

func ApproximateGasLimit(gasUsed uint64) uint64 {
	return uint64(float64(gasUsed) * 1.5)
}

func GetFeeAmount(gasLimit uint64) uint64 {
	return 0
}
