package enviro

import (
	"fmt"
	"github.com/spf13/viper"
	"net"
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
	Number     *[]uint32     `enviro:"number"`
	Host       []string      `enviro:"host"`
	Timeout    time.Duration `enviro:"timeout,required"`
	Integer    int           `enviro:"integer"`
	Time       []time.Time   `enviro:"time" envformat:"time:2006*01*02"`
	Address    []net.IP      `enviro:"address"`
	JsonConfig []JsonConfig  `enviro:"json_config" envformat:"json"`
	// NestedConfig NestedConfig
}

type NestedConfig struct {
	Foo string `enviro:"foo"`
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
	os.Setenv("TEST_HOST", "localhost,foo")
	os.Setenv("TEST_TIME", "2024*03*30,2024*03*31")
	os.Setenv("TEST_JSON_CONFIG", `{"foo":"bar"},{"foo":"baz"}`)
	os.Setenv("TEST_FOO", "baz")
	os.Setenv("TEST_NUMBER", "1,2,3")
	os.Setenv("TEST_ADDRESS", "127.0.0.1,127.0.0.2")

	e := New()
	e.SetEnvPrefix("test")
	cfg := Config{
		Address: []net.IP{{127, 0, 0, 3}},
		Number:  &[]uint32{9},
	}
	if err := e.Load(&cfg); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v\n", cfg)
}

func TestY(t *testing.T) {
	v := viper.New()
	v.SetEnvPrefix("test")
}
