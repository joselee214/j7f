package shard

import (
	"fmt"
	. "github.com/joselee214/j7f/components/dao/errors"
	"hash/crc32"
	"strconv"
	"time"
)

const (
	MIN_LEN_KEY = 1

	MODSHARDTYPE      = "mod"
	RANGESHARDTYPE    = "range"
	DateDayRuleType   = "date_day"
	DateMonthRuleType = "date_month"
	DateYearRuleType  = "date_year"
)

type ShardConfig struct {
	DB            string `yaml:"db"`
	Table         string `yaml:"table"`
	ModNum        int    `yaml:"mod_num"`
	Locations     int    `yaml:"locations"`
	Type          string `yaml:"type"`
	TableRowLimit int    `yaml:"table_row_limit"`
}

type Shard interface {
	FindForKey(key ...interface{}) (int, error)
}

func ModValue(value interface{}) (uint64, error) {
	switch val := value.(type) {
	case int:
		return uint64(val), nil
	case uint64:
		return uint64(val), nil
	case int64:
		return uint64(val), nil
	case string:
		if v, err := strconv.ParseUint(val, 10, 64); err != nil {
			return uint64(crc32.ChecksumIEEE([]byte(val))), nil
		} else {
			return uint64(v), nil
		}
	case []byte:
		return uint64(crc32.ChecksumIEEE(val)), nil
	case []interface{}: //TODO
		if len(val) == 0 {
			return 0, ErrKeyNotExist
		}

	}
	return 0, ErrKeyNotExist
}

func NumValue(value interface{}) (int64, error) {
	switch val := value.(type) {
	case int:
		return int64(val), nil
	case uint64:
		return int64(val), nil
	case int64:
		return int64(val), nil
	case string:
		if v, err := strconv.ParseInt(val, 10, 64); err != nil {
			panic(fmt.Errorf("invalid num format %v", v))
		} else {
			return v, nil
		}
	case []byte:
		if v, err := strconv.ParseInt(string(val), 10, 64); err != nil {
			panic(fmt.Errorf("invalid num format %v", v))
		} else {
			return v, nil
		}
	case []interface{}:
		if len(val) == 0 {
			return 0, ErrKeyNotExist
		}

	}
	return 0, ErrKeyNotExist
}

type ModShard struct {
	ShardNum int
}

func (s *ModShard) FindForKey(key ...interface{}) (int, error) {
	var m uint64
	var err error
	if len(key) == MIN_LEN_KEY {
		m, err = ModValue(key[0])
	} else {
		keys := ""
		for _, v := range key {
			r, err := ModValue(v)
			if err != nil {
				continue
			}
			keys = keys + strconv.Itoa(int(r))
		}
		m, err = ModValue(keys)
	}
	return int(m % uint64(s.ShardNum)), err
}

type NumRangeShard struct {
	Shards []NumKeyRange
}

func (s *NumRangeShard) FindForKey(key ...interface{}) (int, error) {
	var v int64
	var err error
	if len(s.Shards) == 0 {
		return -1, ErrMultiShard
	}
	if len(key) == MIN_LEN_KEY {
		v, err = NumValue(key[0])
	} else {
		keys := ""
		for _, v := range key {
			r, err := NumValue(v)
			if err != nil {
				continue
			}
			keys = keys + strconv.Itoa(int(r))
		}
		v, err = NumValue(keys)
	}
	if err != nil {
		return 0, err
	}
	for i, r := range s.Shards {
		if r.Contains(v) {
			return i, nil
		}
	}
	return -1, ErrKeyOutOfRange
}

func (s *NumRangeShard) EqualStart(key interface{}, index int) (bool, error) {
	v, err := NumValue(key)
	if err != nil {
		return false, err
	}
	return s.Shards[index].Start == v, nil
}
func (s *NumRangeShard) EqualStop(key interface{}, index int) (bool, error) {
	v, err := NumValue(key)
	if err != nil {
		return false, nil
	}
	return s.Shards[index].End == v, nil
}

type DateYearShard struct {
}

//the format of date is: YYYY-MM-DD HH:MM:SS,YYYY-MM-DD or unix timestamp(int)
func (s *DateYearShard) FindForKey(key ...interface{}) (int, error) {
	keyT := key[0]
	switch val := keyT.(type) {
	case int:
		tm := time.Unix(int64(val), 0)
		return tm.Year(), nil
	case uint64:
		tm := time.Unix(int64(val), 0)
		return tm.Year(), nil
	case int64:
		tm := time.Unix(val, 0)
		return tm.Year(), nil
	case string:
		if v, err := strconv.Atoi(val[:4]); err != nil {
			return 0, ErrKeyNotExist
		} else {
			return v, nil
		}
	}
	return 0, ErrKeyNotExist
}

type DateMonthShard struct {
}

//the format of date is: YYYY-MM-DD HH:MM:SS,YYYY-MM-DD or unix timestamp(int)
func (s *DateMonthShard) FindForKey(key ...interface{}) (int, error) {
	keyT := key[0]
	timeFormat := "2006-01-02"
	switch val := keyT.(type) {
	case int:
		tm := time.Unix(int64(val), 0)
		dateStr := tm.Format(timeFormat)
		s := dateStr[:4] + dateStr[5:7]
		yearMonth, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return yearMonth, nil
	case uint64:
		tm := time.Unix(int64(val), 0)
		dateStr := tm.Format(timeFormat)
		s := dateStr[:4] + dateStr[5:7]
		yearMonth, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return yearMonth, nil
	case int64:
		tm := time.Unix(val, 0)
		dateStr := tm.Format(timeFormat)
		s := dateStr[:4] + dateStr[5:7]
		yearMonth, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return yearMonth, nil
	case string:
		if len(val) < len(timeFormat) {
			return 0, fmt.Errorf("invalid date format %s", val)
		}
		s := val[:4] + val[5:7]
		if v, err := strconv.Atoi(s); err != nil {
			return 0, fmt.Errorf("invalid date format %s", val)
		} else {
			return v, nil
		}
	}
	return 0, ErrKeyNotExist
}

type DateDayShard struct {
}

//the format of date is: YYYY-MM-DD HH:MM:SS,YYYY-MM-DD or unix timestamp(int)
func (s *DateDayShard) FindForKey(key ...interface{}) (int, error) {
	keyT := key[0]
	timeFormat := "2006-01-02"
	switch val := keyT.(type) {
	case int:
		tm := time.Unix(int64(val), 0)
		dateStr := tm.Format(timeFormat)
		s := dateStr[:4] + dateStr[5:7] + dateStr[8:10]
		yearMonthDay, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return yearMonthDay, nil
	case uint64:
		tm := time.Unix(int64(val), 0)
		dateStr := tm.Format(timeFormat)
		s := dateStr[:4] + dateStr[5:7] + dateStr[8:10]
		yearMonthDay, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return yearMonthDay, nil
	case int64:
		tm := time.Unix(val, 0)
		dateStr := tm.Format(timeFormat)
		s := dateStr[:4] + dateStr[5:7] + dateStr[8:10]
		yearMonthDay, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return yearMonthDay, nil
	case string:
		if len(val) < len(timeFormat) {
			return 0, fmt.Errorf("invalid date format %s", val)
		}
		s := val[:4] + val[5:7] + val[8:10]
		if v, err := strconv.Atoi(s); err != nil {
			return 0, fmt.Errorf("invalid date format %s", val)
		} else {
			return v, nil
		}
	}
	return 0, ErrKeyNotExist
}

func ParseShard(cfgs []*ShardConfig) ([]Shard, error) {
	shards := make([]Shard, 0)
	for _, cfg := range cfgs {
		switch cfg.Type {
		case MODSHARDTYPE:
			shard := &ModShard{
				ShardNum: cfg.ModNum,
			}
			shards = append(shards, shard)
		case RANGESHARDTYPE:
			rs, err := ParseNumSharding(cfg.Locations, cfg.TableRowLimit)
			if err != nil {
				return nil, err
			}
			shard := &NumRangeShard{
				Shards: rs,
			}
			shards = append(shards, shard)
		case DateDayRuleType:
			shard := &DateDayShard{}
			shards = append(shards, shard)
		case DateMonthRuleType:
			shard := &DateMonthShard{}
			shards = append(shards, shard)
		case DateYearRuleType:
			shard := &DateYearShard{}
			shards = append(shards, shard)
		}
	}

	return shards, nil
}
