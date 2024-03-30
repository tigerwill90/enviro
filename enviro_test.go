package enviro

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"testing"
	"time"
)

type Config struct {
	Timeout    *time.Duration `enviro:"timeout,required"`
	Host       *string        `enviro:"host,required"`
	Port       *uint          `enviro:"port"`
	Time       *time.Time     `enviro:"time" envformat:"time:2006*01*02,Europe/Berlin"`
	JsonConfig *JsonConfig    `enviro:"json_config" envformat:"json"`
}

type JsonConfig struct {
	Foo string `json:"foo"`
}

func TestX(t *testing.T) {
	os.Setenv("TEST_TIMEOUT", "10s")
	os.Setenv("TEST_HOST", "localhost")
	os.Setenv("TEST_PORT", "8080")
	os.Setenv("TEST_TIME", "2024*03*30")
	os.Setenv("TEST_JSON_CONFIG", `{"foo":"bar"}`)

	e := New()
	e.SetEnvPrefix("test")
	cfg := Config{}
	if err := e.Load(&cfg); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v\n", cfg)
}

func TestY(t *testing.T) {
	v := viper.New()
	v.SetEnvPrefix("test")
}
