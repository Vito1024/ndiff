package ndiff

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	SERVICE_NAME = "nft_diff"
	TIME_FORMAT  = time.RFC3339

	DEFAULT_START_HEIGHT = uint64(21000)
)

func init() {
	parseEnv()
}

var (
	LOG_LEVEL = LogLevel(os.Getenv("LOG_LEVEL"))

	startHeight = os.Getenv("START_HEIGHT")
	endHeight   = os.Getenv("END_HEIGHT")
	step        = os.Getenv("STEP")

	DIFF_RESULT_FILE_LOCATION = os.Getenv("DIFF_RESULT_FILE_LOCATION")

	START_HEIGHT uint64
	END_HEIGHT   uint64
	STEP         uint64
)

type LogLevel string

const (
	LOG_LEVEL_DEBUG LogLevel = "debug"
	LOG_LEVEL_INFO  LogLevel = "info"
)

func parseEnv() {
	if endHeight == "" {
		panic("END_HEIGHT is not set")
	}
	if startHeight == "" {
		startHeight = "21000"
	}

	var err error
	START_HEIGHT, err = strconv.ParseUint(startHeight, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("failed to parse START_HEIGHT, err: %v", err))
	}
	END_HEIGHT, err = strconv.ParseUint(endHeight, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("failed to parse END_HEIGHT, err: %v", err))
	}
	if START_HEIGHT < DEFAULT_START_HEIGHT {
		START_HEIGHT = DEFAULT_START_HEIGHT
	}

	if !(END_HEIGHT >= START_HEIGHT) {
		panic("END_HEIGHT must be greater than START_HEIGHT")
	}

	if step == "" {
		step = "100"
	}
	STEP, err = strconv.ParseUint(step, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("failed to parse STEP, err: %v", err))
	}
	if STEP == 0 {
		panic("STEP must be greater than 0")
	}
}

type Tracker interface {
	// basic log
	Debug(eventType string, message string, kv ...Tag)
	Info(eventType string, message string, kv ...Tag)
	Warn(eventType string, message string, kv ...Tag)
	Error(eventType string, message string, kv ...Tag)
	Fatal(eventType string, message string, kv ...Tag)

	Flush()
}

type Tag struct {
	Key   string
	Value interface{}
}

func NewTag(key string, value interface{}) Tag {
	str, ok := value.(fmt.Stringer)
	if ok {
		return Tag{Key: key, Value: str.String()}
	}
	return Tag{Key: key, Value: fmt.Sprintf("%+v", value)}
}

func ErrorTag(err error) Tag {
	return Tag{Key: "error", Value: err}
}
