package enviro

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/spf13/viper"
	"os"
	"testing"
	"time"
)

type Config struct {
	/*	Timeout      time.Duration `enviro:"timeout,required"`
		Host         string        `enviro:"host,required"`
		Port         uint          `enviro:"port"`
		Time         time.Time     `enviro:"time" envformat:"time:2006*01*02,Europe/Berlin"`
		JsonConfig   *JsonConfig   `enviro:"json_config" envformat:"json"`*/
	Host         string                 `enviro:"host,required"`
	Bytes        []BytesSize            `enviro:"bytes"`
	JsonConfig   map[string]interface{} `enviro:"json_config" envformat:"json"`
	NestedConfig NestedConfig           `enviro:"prefix:my_config"`
}

type Employees struct {
	Employees []Employee `yaml:"employees"`
}

type Employee struct {
	ID         int    `yaml:"id"`
	Name       string `yaml:"name"`
	Role       string `yaml:"role"`
	Department string `yaml:"department"`
}

type NestedConfig struct {
	Foo string        `enviro:"foo"`
	Baz NestedConfig2 `enviro:"prefix:and"`
}

type NestedConfig2 struct {
	Bar string `enviro:"bar"`
}

type DurationAlias time.Duration

func (d DurationAlias) String() string {
	return time.Duration(d).String()
}

type TimeAlias time.Time

func (t TimeAlias) String() string {
	return time.Time(t).String()
}

type JsonConfig struct {
	Foo string `json:"foo"`
}

func TestX(t *testing.T) {
	os.Setenv("TEST_TIMEOUT", "10s")
	os.Setenv("TEST_HOST", "localhost")
	os.Setenv("TEST_TIME", "2024*03*30,2024*03*31")
	os.Setenv("TEST_JSON_CONFIG", `{"foo": "bar"}`)
	os.Setenv("TEST_MY_CONFIG_FOO", "foo")
	os.Setenv("TEST_MY_CONFIG_AND_BAR", "bar")
	os.Setenv("TEST_NUMBER", "1,2,3")
	os.Setenv("TEST_ADDRESS", "127.0.0.1,127.0.0.2")
	os.Setenv("TEST_LOCATION", "UTC")
	os.Setenv("TEST_BYTES", "10Mb")

	e := New()
	e.SetEnvPrefix("test")
	cfg := Config{}
	if err := e.ParseEnv(&cfg); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v\n", cfg)
}

func TestY(t *testing.T) {
	v := viper.New()
	v.SetEnvPrefix("test")

	var b BytesSize
	// fmt.Println(b.ParseField(""))
	fmt.Println(b)
}

type BytesSize uint64

func (b *BytesSize) ParseField(value string) error {
	f, err := humanize.ParseBytes(value)
	if err != nil {
		return err
	}
	*b = BytesSize(f)
	return nil
}

func (b BytesSize) String() string {
	return humanize.Bytes(uint64(b))
}
