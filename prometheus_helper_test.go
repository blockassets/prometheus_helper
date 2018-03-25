package prometheus_helper

import (
	"testing"
)

type MapInt64 map[string]int64

type ChipStat struct {
	DevCommon
	Accept  MapInt64
	Funny   int64  `json:"Funny"`
	Ignored string `json:"Ignored,omit"`
}

type DevCommon struct {
	Name    string `json:"Name,string"`
	ASC     int    `json:"ASC"`
	ID      int    `json:"ID"`
	BigName int    `json:"This is a big name"`
	Jon     int
}

func TestMakeStructMeta(t *testing.T) {
	meta := StructMeta{}
	chipStat := ChipStat{
		DevCommon: DevCommon{
			Name: "ttsy1",
			Jon:  456,
		},
		Accept:  MapInt64{"Jon": 123, "mat": 567},
		Funny:   1234,
		Ignored: "not here",
	}

	MakeStructMeta(chipStat, &meta)

	if len(meta.Data) != 7 {
		t.Errorf("Length is wrong, expected 7 and got %v", len(meta.Data))
	}

	if meta.Data["Name"].Value != "ttsy1" {
		t.Errorf("Value is wrong, expected ttsy1 and got %s", meta.Data["Name"].Value)
	}

	if meta.Data["Jon"].Value != int64(123) {
		t.Errorf("Value is wrong, expected 123 and got %s", meta.Data["Jon"].Value)
	}

	if meta.Data["BigName"].Tag != "This is a big name" {
		t.Errorf("Value is wrong, expected 'This is a big name' and got %s", meta.Data["BigName"].Tag)
	}
}

func TestNewGaugeMap(t *testing.T) {
	chipStat := ChipStat{
		DevCommon: DevCommon{
			Name: "ttsy1",
			Jon:  456,
		},
		Accept: MapInt64{"Jon": 123, "mat": 567, "funky %": 999, "1_accept": 3333},
		Funny:  1234,
	}
	meta := StructMeta{}
	MakeStructMeta(chipStat, &meta)

	result := NewGaugeMap(meta, "jon", nil)
	if _, ok := result["jon_chipstat_1_accept"]; !ok {
		t.Error("Key not found: jon_chipstat_1_accept")
	}
	if _, ok := result["jon_chipstat_this_is_a_big_name"]; !ok {
		t.Error("Key not found: jon_chipstat_this_is_a_big_name")
	}
	if _, ok := result["jon_chipstat_funky_percent"]; !ok {
		t.Error("Key not found: jon_chipstat_funky_percent")
	}
}

func TestNewGaugeVecMap(t *testing.T) {
	chipStat := ChipStat{
		DevCommon: DevCommon{
			Name: "ttsy1",
			Jon:  456,
		},
		Accept: MapInt64{"Jon": 123, "mat": 567, "funky %": 999, "1_accept": 3333},
		Funny:  1234,
	}
	meta := StructMeta{}
	MakeStructMeta(chipStat, &meta)

	result := NewGaugeVecMap(meta, "jon", nil, nil)
	if _, ok := result["jon_chipstat_1_accept"]; !ok {
		t.Error("Key not found: jon_chipstat_1_accept")
	}
	if _, ok := result["jon_chipstat_this_is_a_big_name"]; !ok {
		t.Error("Key not found: jon_chipstat_this_is_a_big_name")
	}
	if _, ok := result["jon_chipstat_funky_percent"]; !ok {
		t.Error("Key not found: jon_chipstat_funky_percent")
	}
}

type PoolData struct {
	User    string
	Workers string
}

func TestNewStructFieldMap(t *testing.T) {
	pd := PoolData{
		User:    "jon",
		Workers: "bar",
	}

	response := NewStructFieldMap(pd)
	if _, ok := response["User"]; !ok {
		t.Error("User not found in PoolData")
	}
}

func TestConvertToFloat(t *testing.T) {
	val, err := ConvertToFloat("Y")
	if err != nil || val != 1 {
		t.Fatalf("Failed to convert Y to 1")
	}

	val, err = ConvertToFloat("y")
	if err != nil || val != 1 {
		t.Fatalf("Failed to convert y to 1")
	}

	val, err = ConvertToFloat("n")
	if err != nil || val != 0 {
		t.Fatalf("Failed to convert n to 0")
	}

	val, err = ConvertToFloat("Alive")
	if err != nil || val != 1 {
		t.Fatalf("Failed to convert Alive to 1")
	}

	val, err = ConvertToFloat(true)
	if err != nil || val != 1 {
		t.Fatalf("Failed to convert true to 1")
	}
}
