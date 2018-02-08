package main

import (
	"reflect"
	"strings"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Makefile build
	version = ""
)

type StructMeta struct {
	Name string
	Data map[string]TagValue
}

type TagValue struct {
	Tag string // The parsed tag string `json:"Foo,string"` == Foo
	Value interface{} // The value of the field in the struct
}

/*
	Recursively follow a nested tree of struct and add the names
	of the fields to the names map. If the field value is a map,
	use the keys from that as the names. If there is a name conflict
	then we use the last one found and throw a warning message.

	Check the test for an example.
*/
func MakeStructMeta(strct interface{}, meta *StructMeta) {
	val := reflect.ValueOf(strct)

	if len(meta.Name) == 0 {
		meta.Name = val.Type().Name()
		meta.Data = make(map[string]TagValue)
	}

	numFields := val.NumField()

	for i := 0; i < numFields; i++ {
		field := val.Field(i)
		fieldKind := field.Kind()
		fieldInterface := field.Interface()

		if fieldKind == reflect.Struct {
			MakeStructMeta(fieldInterface, meta)
		} else {
			structField := val.Type().Field(i)
			tag := strings.Split(structField.Tag.Get("json"), ",")[0]

			tagValue := TagValue{
				Tag: tag,
			}

			fieldValue := reflect.ValueOf(fieldInterface)
			if fieldKind == reflect.Map {
				for _, key := range fieldValue.MapKeys() {
					tagValue.Value = fieldValue.MapIndex(key).Interface()
					keyString := key.String()
					if _, ok := meta.Data[keyString]; ok {
						fmt.Printf("Warning! key: %s exists already in Map, replacing with value %v\n", keyString, tagValue.Value)
					}
					meta.Data[keyString] = tagValue
				}
			} else {
				tagValue.Value = fieldInterface
				if _, ok := meta.Data[structField.Name]; ok {
					fmt.Printf("Warning! key: %s exists already, replacing with value %v\n", structField.Name, tagValue.Value)
				}
				meta.Data[structField.Name] = tagValue
			}
		}
	}
}

func fixName(tag string) string {
	// Order matters
	r := strings.Replace(tag, " %", " percent", -1)
	r = strings.Replace(r, "%", "_percent", -1)
	r = strings.Replace(r, " ", "_", -1)
	return r
}

//
func fmtGaugeName(interfaceName string, tag string) string {
	return fmt.Sprintf("%s_%s", strings.ToLower(interfaceName), strings.ToLower(fixName(tag)))
}

//
func fmtGaugeDescription(interfaceName string, tag string) string {
	return fmt.Sprintf("%s%s", interfaceName, tag)
}

//
func NewGauge(namespace string, metricName string, help string, constLabels prometheus.Labels) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        metricName,
		Help:        help,
		ConstLabels: constLabels,
	})
}

//
func NewGaugeVec(namespace string, metricName string, help string, constLabels prometheus.Labels, labels []string) prometheus.GaugeVec {
	return *prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        metricName,
			Help:        help,
			ConstLabels: constLabels,
		},
		labels,
	)
}

type GaugeMap map[string]prometheus.Gauge
type GaugeVecMap map[string]prometheus.GaugeVec

/*
	Builds a map of Gauges, the names are built out of the struct property names.
	Looking first to see if there is a json tag name and if not, grab an actual name.
 */
func NewGaugeMap(meta StructMeta, namespace string, constLabels prometheus.Labels) GaugeMap {
	metrics := GaugeMap{}

	for key, value := range meta.Data {

		// Try to use the tag name first
		baseName := value.Tag
		if len(baseName) == 0 {
			baseName = key
		}

		gaugeName := fmtGaugeName(meta.Name, baseName)

		gauge := NewGauge(
			namespace,
			gaugeName,
			fmtGaugeDescription(meta.Name, key),
			constLabels)

		metrics[namespace + "_" + gaugeName] = gauge
	}

	return metrics
}

/*
	Builds a map of Gauges, the names are built out of the struct property names.
	Looking first to see if there is a json tag name and if not, grab an actual name.
 */
func NewGaugeVecMap(meta StructMeta, namespace string, labels []string, constLabels prometheus.Labels) GaugeVecMap {
	metrics := GaugeVecMap{}

	for key, value := range meta.Data {
		// Try to use the tag name first
		baseName := value.Tag
		if len(baseName) == 0 {
			baseName = key
		}

		gaugeName := fmtGaugeName(meta.Name, baseName)

		gauge := NewGaugeVec(
			namespace,
			gaugeName,
			fmtGaugeDescription(meta.Name, key),
			constLabels, labels)

		metrics[namespace + "_" + gaugeName] = gauge
	}

	return metrics
}
