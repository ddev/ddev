package constants

import (
	"time"

	"github.com/amplitude/analytics-go/amplitude/types"
)

const (
	SdkLibrary = "amplitude-go"
	SdkVersion = "1.0.1"

	IdentifyEventType      = "$identify"
	GroupIdentifyEventType = "$groupidentify"
	RevenueEventType       = "revenue_amount"

	LoggerName = "amplitude"

	RevenueProductID  = "$productId"
	RevenueQuantity   = "$quantity"
	RevenuePrice      = "$price"
	RevenueType       = "$revenueType"
	RevenueReceipt    = "$receipt"
	RevenueReceiptSig = "$receiptSig"
	DefaultRevenue    = "$revenue"

	MaxPropertyKeys = 1024
	MaxStringLength = 1024
)

var ServerURLs = map[types.ServerZone]string{
	types.ServerZoneUS: "https://api2.amplitude.com/2/httpapi",
	types.ServerZoneEU: "https://api.eu.amplitude.com/2/httpapi",
}

var ServerBatchURLs = map[types.ServerZone]string{
	types.ServerZoneUS: "https://api2.amplitude.com/batch",
	types.ServerZoneEU: "https://api.eu.amplitude.com/batch",
}

var DefaultConfig = types.Config{
	FlushInterval:          time.Second * 10,
	FlushQueueSize:         200,
	FlushSizeDivider:       1,
	FlushMaxRetries:        12,
	ServerZone:             types.ServerZoneUS,
	ConnectionTimeout:      time.Second * 10,
	MaxStorageCapacity:     20000,
	RetryBaseInterval:      time.Millisecond * 100,
	RetryThrottledInterval: time.Second * 30,
}
