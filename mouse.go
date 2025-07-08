package main

const (
	LAMZU_VID        = 0x373E
	LAMZU_PID        = 0x001E
	INTERFACE_NUMBER = 2
	REPORT_SIZE      = 65
)

var pollingRateMap = map[int]byte{
	500:  2,
	1000: 1,
	2000: 32,
	4000: 64,
	8000: 128,
}

type MouseControllerInterface interface {
	Close()
	TestConnection() error
	SetPollingRate(rate int) error
}

func parsePollingRate(s string) int {
	switch s {
	case "500":
		return 500
	case "1000":
		return 1000
	case "2000":
		return 2000
	case "4000":
		return 4000
	case "8000":
		return 8000
	default:
		return 0
	}
}
