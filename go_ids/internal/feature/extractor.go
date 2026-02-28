package feature

import (
	"go-ids/internal/flow"
)

// Extractor 负责从 flow.Flow 对象中提取 78 个特征
type Extractor struct{}

// NewExtractor 创建一个新的特征提取器
func NewExtractor() *Extractor {
	return &Extractor{}
}

// Extract 从流中提取特征向量
func (e *Extractor) Extract(f *flow.Flow) []float32 {
	features := make([]float32, 78)

	// 1. Destination Port
	features[0] = float32(f.Key.DstPort)

	// 2. Flow Duration
	duration := f.LastTime.Sub(f.StartTime).Seconds() * 1000000 // 微秒
	features[1] = float32(duration)

	// 3. Total Fwd Packets
	features[2] = float32(f.FwdPackets)

	// 4. Total Backward Packets
	features[3] = float32(f.BwdPackets)

	// 5. Total Length of Fwd Packets
	features[4] = float32(f.FwdBytes)

	// 6. Total Length of Bwd Packets
	features[5] = float32(f.BwdBytes)

	// 7-10. Fwd Packet Length (Max, Min, Mean, Std)
	features[6] = float32(f.FwdPktLenMax)
	features[7] = sanitizeFloat(f.FwdPktLenMin)
	features[8] = float32(flow.GetMean(f.FwdPktLenSum, f.FwdPackets))
	features[9] = float32(flow.GetStd(f.FwdPktLenSum, f.FwdPktLenSqSum, f.FwdPackets))

	// 11-14. Bwd Packet Length (Max, Min, Mean, Std)
	features[10] = float32(f.BwdPktLenMax)
	features[11] = sanitizeFloat(f.BwdPktLenMin)
	features[12] = float32(flow.GetMean(f.BwdPktLenSum, f.BwdPackets))
	features[13] = float32(flow.GetStd(f.BwdPktLenSum, f.BwdPktLenSqSum, f.BwdPackets))

	// 15. Flow Bytes/s
	if duration > 0 {
		features[14] = float32(float64(f.FwdBytes+f.BwdBytes) / (duration / 1000000.0))
	} else {
		features[14] = 0
	}

	// 16. Flow Packets/s
	if duration > 0 {
		features[15] = float32(float64(f.FwdPackets+f.BwdPackets) / (duration / 1000000.0))
	} else {
		features[15] = 0
	}

	// 17-20. Flow IAT (Mean, Std, Max, Min)
	totalPkts := f.FwdPackets + f.BwdPackets
	iatCount := uint64(0)
	if totalPkts > 1 {
		iatCount = totalPkts - 1
	}
	features[16] = float32(flow.GetMean(f.FlowIATSum, iatCount))
	features[17] = float32(flow.GetStd(f.FlowIATSum, f.FlowIATSqSum, iatCount))
	features[18] = float32(f.FlowIATMax)
	features[19] = sanitizeFloat(f.FlowIATMin)

	// 21-25. Fwd IAT (Total, Mean, Std, Max, Min)
	fwdIATCount := uint64(0)
	if f.FwdPackets > 1 {
		fwdIATCount = f.FwdPackets - 1
	}
	features[20] = float32(f.FwdIATSum)
	features[21] = float32(flow.GetMean(f.FwdIATSum, fwdIATCount))
	features[22] = float32(flow.GetStd(f.FwdIATSum, f.FwdIATSqSum, fwdIATCount))
	features[23] = float32(f.FwdIATMax)
	features[24] = sanitizeFloat(f.FwdIATMin)

	// 26-30. Bwd IAT (Total, Mean, Std, Max, Min)
	bwdIATCount := uint64(0)
	if f.BwdPackets > 1 {
		bwdIATCount = f.BwdPackets - 1
	}
	features[25] = float32(f.BwdIATSum)
	features[26] = float32(flow.GetMean(f.BwdIATSum, bwdIATCount))
	features[27] = float32(flow.GetStd(f.BwdIATSum, f.BwdIATSqSum, bwdIATCount))
	features[28] = float32(f.BwdIATMax)
	features[29] = sanitizeFloat(f.BwdIATMin)

	// 31-34. PSH/URG Flags
	features[30] = float32(f.FwdPSHFlags)
	features[31] = float32(f.BwdPSHFlags)
	features[32] = float32(f.FwdURGFlags)
	features[33] = float32(f.BwdURGFlags)

	// 35-36. Header Length
	features[34] = float32(f.FwdHeaderLen)
	features[35] = float32(f.BwdHeaderLen)

	// 37-38. Packets/s (Fwd, Bwd)
	if duration > 0 {
		features[36] = float32(float64(f.FwdPackets) / (duration / 1000000.0))
		features[37] = float32(float64(f.BwdPackets) / (duration / 1000000.0))
	} else {
		features[36], features[37] = 0, 0
	}

	// 39-43. Packet Length Stats (Min, Max, Mean, Std, Var)
	features[38] = sanitizeFloat(f.PktLenMin)
	features[39] = float32(f.PktLenMax)
	features[40] = float32(flow.GetMean(f.PktLenSum, totalPkts))
	pktStd := flow.GetStd(f.PktLenSum, f.PktLenSqSum, totalPkts)
	features[41] = float32(pktStd)
	features[42] = float32(pktStd * pktStd)

	// 44-51. TCP Flag Counts
	features[43] = float32(f.FINFlagCount)
	features[44] = float32(f.SYNFlagCount)
	features[45] = float32(f.RSTFlagCount)
	features[46] = float32(f.PSHFlagCount)
	features[47] = float32(f.ACKFlagCount)
	features[48] = float32(f.URGFlagCount)
	features[49] = float32(f.CWEFlagCount)
	features[50] = float32(f.ECEFlagCount)

	// 52. Down/Up Ratio
	if f.FwdPackets > 0 {
		features[51] = float32(f.BwdPackets) / float32(f.FwdPackets)
	} else {
		features[51] = 0
	}

	// 53. Average Packet Size
	if totalPkts > 0 {
		features[52] = float32(f.PktLenSum) / float32(totalPkts)
	} else {
		features[52] = 0
	}

	// 54-55. Avg Segment Size (Mean of packet lengths)
	features[53] = float32(flow.GetMean(f.FwdPktLenSum, f.FwdPackets))
	features[54] = float32(flow.GetMean(f.BwdPktLenSum, f.BwdPackets))

	// 56. Fwd Header Length.1 (Same as Fwd Header Length)
	features[55] = float32(f.FwdHeaderLen)

	// 57-62. Bulk related (Simplified to 0)
	features[56], features[57], features[58] = 0, 0, 0
	features[59], features[60], features[61] = 0, 0, 0

	// 63-66. Subflow (Often same as Total in short duration or single flow)
	features[62] = float32(f.FwdPackets)
	features[63] = float32(f.FwdBytes)
	features[64] = float32(f.BwdPackets)
	features[65] = float32(f.BwdBytes)

	// 67-68. Init Win Bytes
	features[66] = float32(f.InitWinBytesFwd)
	features[67] = float32(f.InitWinBytesBwd)

	// 69. Act Data Pkts Fwd
	features[68] = float32(f.FwdActDataPkts)

	// 70. Min Seg Size Fwd
	features[69] = float32(f.FwdMinSegSize)

	// 71-78. Active/Idle (Simplified)
	features[70] = float32(f.ActiveSum)
	features[71] = float32(flow.GetStd(f.ActiveSum, f.ActiveSqSum, 1)) // 简化
	features[72] = float32(f.ActiveMax)
	features[73] = sanitizeFloat(f.ActiveMin)
	features[74] = float32(f.IdleSum)
	features[75] = float32(flow.GetStd(f.IdleSum, f.IdleSqSum, 1)) // 简化
	features[76] = float32(f.IdleMax)
	features[77] = sanitizeFloat(f.IdleMin)

	return features
}

// sanitizeFloat 处理极值，防止 1e9 等初始化值污染特征
func sanitizeFloat(val float64) float32 {
	if val >= 1e9 || val < 0 {
		return 0
	}
	return float32(val)
}
