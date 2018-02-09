package prometheus_helper

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type StructMeta struct {
	Name string
	Data map[string]TagValue
}

/*
	Key: FieldName
	Value: FieldValue

	type PoolData struct {
		User    User    `json:"user"`
		Workers Workers `json:"workers"`
		Pool    Pool    `json:"pool"`
		Network Network `json:"network"`
		Market  Market  `json:"market"`
	}

	map["User"] = User
	map["Workers"] = Workers
	...
 */
type StructFieldMap map[string]interface{}

type TagValue struct {
	Tag   string      // The parsed tag string `json:"Foo,string"` == Foo
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

	// Skip any Maps passed in
	if val.Kind() == reflect.Map {
		return
	}

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

func MakeStructFieldMap(strct interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	val := reflect.ValueOf(strct)
	numFields := val.NumField()

	for i := 0; i < numFields; i++ {
		field := val.Field(i)
		fieldInterface := field.Interface()
		structField := val.Type().Field(i)

		result[structField.Name] = fieldInterface
	}

	return result
}

func fixName(tag string) string {
	// Order matters
	r := strings.Replace(tag, " %", " percent", -1)
	r = strings.Replace(r, "%", "_percent", -1)
	r = strings.Replace(r, " ", "_", -1)
	return r
}

//
func fmtGaugeName(interfaceName string, key string, value TagValue) string {
	// Try to use the tag name first as that tends to produce more attractive names
	baseName := value.Tag
	if len(baseName) == 0 {
		baseName = key
	}
	return fmt.Sprintf("%s_%s", strings.ToLower(interfaceName), strings.ToLower(fixName(baseName)))
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
// I must be insane.
type GaugeMapMap map[string]GaugeMap
type GaugeVecMapMap map[string]GaugeVecMap


/*
	Prometheus won't start up if there are no Gauges defined, so provide a helper to build the MapMap.
 */
func NewGaugeMapMap(faces map[string]interface{}, namespace string, constLabels prometheus.Labels) GaugeMapMap {
	gmm := make(GaugeMapMap)
	for _, iface := range faces {
		meta := StructMeta{}
		MakeStructMeta(iface, &meta)
		if _, ok := gmm[meta.Name]; !ok {
			gmm[meta.Name] = NewGaugeMap(meta, namespace, constLabels)
		}
	}
	return gmm
}

/*
	Builds a map of Gauges, the names are built out of the struct property names.
	Looking first to see if there is a json tag name and if not, grab an actual name.
*/
func NewGaugeMap(meta StructMeta, namespace string, constLabels prometheus.Labels) GaugeMap {
	metrics := GaugeMap{}

	for key, value := range meta.Data {

		gaugeName := fmtGaugeName(meta.Name, key, value)

		gauge := NewGauge(
			namespace,
			gaugeName,
			fmtGaugeDescription(meta.Name, key),
			constLabels)

		metrics[namespace+"_"+gaugeName] = gauge
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

		gaugeName := fmtGaugeName(meta.Name, key, value)

		gauge := NewGaugeVec(
			namespace,
			gaugeName,
			fmtGaugeDescription(meta.Name, key),
			constLabels, labels)

		metrics[namespace+"_"+gaugeName] = gauge
	}

	return metrics
}

func SetValuesOnGauges(meta StructMeta, namespace string, gaugeMap GaugeMap) {
	for key, value := range meta.Data {
		flt, err := ConvertToFloat(value.Value)
		if err != nil {
			fmt.Println("Error converting %s->%s : %v to float", meta.Name, key, value.Value)
		} else {
			gaugeName := fmtGaugeName(meta.Name, key, value)
			if gauge, ok := gaugeMap[namespace+"_"+gaugeName]; ok {
				gauge.Set(flt)
			}
		}
	}
}

func SetValuesOnGaugeVecs(meta StructMeta, namespace string, gaugeVecMap GaugeVecMap, labels prometheus.Labels) {
	for key, value := range meta.Data {
		flt, err := ConvertToFloat(value.Value)
		if err != nil {
			fmt.Println("Error converting %s->%s : %v to float", meta.Name, key, value.Value)
		} else {
			gaugeName := fmtGaugeName(meta.Name, key, value)
			if gauge, ok := gaugeVecMap[namespace+"_"+gaugeName]; ok {
				gauge.With(labels).Set(flt)
			}
		}
	}
}

//
func CollectGaugeMap(gaugeMap GaugeMap, ch chan<- prometheus.Metric) {
	for _, metric := range gaugeMap {
		metric.Collect(ch)
	}
}

//
func CollectGaugeMapMap(gaugeMapMap GaugeMapMap, ch chan<- prometheus.Metric) {
	for _, gaugeMap := range gaugeMapMap {
		for _, metric := range gaugeMap {
			metric.Collect(ch)
		}
	}
}

//
func CollectGaugeVecMapMap(gaugeVecMapMap GaugeVecMapMap, ch chan<- prometheus.Metric) {
	for _, gaugeVecMap := range gaugeVecMapMap {
		for _, metric := range gaugeVecMap {
			metric.Collect(ch)
		}
	}
}

//
func DescribeGaugeMap(gaugeMap GaugeMap, ch chan<- *prometheus.Desc) {
	for _, metric := range gaugeMap {
		metric.Describe(ch)
	}
}

//
func DescribeGaugeMapMap(gaugeMapMap GaugeMapMap, ch chan<- *prometheus.Desc) {
	for _, gaugeMap := range gaugeMapMap {
		for _, metric := range gaugeMap {
			metric.Describe(ch)
		}
	}
}

//
func DescribeGaugeVecMapMap(gaugeVecMapMap GaugeVecMapMap, ch chan<- *prometheus.Desc) {
	for _, gaugeVecMap := range gaugeVecMapMap {
		for _, metric := range gaugeVecMap {
			metric.Describe(ch)
		}
	}
}

var floatType = reflect.TypeOf(float64(0))

func ConvertToFloat(unk interface{}) (float64, error) {
	v := reflect.ValueOf(unk)
	v = reflect.Indirect(v)

	// Why not supported by the language?
	if v.Type().Name() == "bool" {
		if v.Bool() {
			return 1, nil
		} else {
			return 0, nil
		}
	}

	if !v.Type().ConvertibleTo(floatType) {
		return 0, fmt.Errorf("cannot convert %v to float64", v.Type())
	}
	fv := v.Convert(floatType)
	return fv.Float(), nil
}
