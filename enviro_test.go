// Copyright 2024 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT License that can be found
// at https://github.com/tigerwill90/enviro/blob/master/LICENSE.txt.

package enviro

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestParseEnvSimple(t *testing.T) {
	type Config struct {
		Name      string `enviro:"name"`
		Age       int    `enviro:"age"`
		IsMarried bool   `enviro:"is_married"`
	}

	os.Setenv("NAME", "John Doe")
	os.Setenv("AGE", "30")
	os.Setenv("IS_MARRIED", "true")
	defer func() {
		os.Unsetenv("NAME")
		os.Unsetenv("AGE")
		os.Unsetenv("IS_MARRIED")
	}()

	expected := Config{
		Name:      "John Doe",
		Age:       30,
		IsMarried: true,
	}

	var config Config
	e := New()
	if err := e.ParseEnv(&config); err != nil {
		t.Errorf("Failed to parse environment variables: %s", err)
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("Expected %+v, got %+v", expected, config)
	}
}

func TestParseEnvNestedWithoutPrefix(t *testing.T) {
	type Address struct {
		City  string `enviro:"city"`
		State string `enviro:"state"`
	}
	type Person struct {
		Name    string `enviro:"name"`
		Address Address
	}

	os.Setenv("NAME", "John Doe")
	os.Setenv("CITY", "New York")
	os.Setenv("STATE", "NY")
	defer func() {
		os.Unsetenv("NAME")
		os.Unsetenv("CITY")
		os.Unsetenv("STATE")
	}()

	expected := Person{
		Name: "John Doe",
		Address: Address{
			City:  "New York",
			State: "NY",
		},
	}

	var person Person
	e := New()
	if err := e.ParseEnv(&person); err != nil {
		t.Errorf("Failed to parse nested environment variables: %s", err)
	}

	if !reflect.DeepEqual(person, expected) {
		t.Errorf("Expected %+v, got %+v", expected, person)
	}
}

func TestParseEnvNestedWithPrefix(t *testing.T) {
	type Address struct {
		City  string `enviro:"city"`
		State string `enviro:"state"`
	}
	type Person struct {
		Name    string  `enviro:"name"`
		Address Address `enviro:"nested:address"`
	}

	// Set environment variables for the test
	os.Setenv("NAME", "John Doe")
	os.Setenv("ADDRESS_CITY", "New York")
	os.Setenv("ADDRESS_STATE", "NY")
	defer func() {
		// Cleanup environment variables
		os.Unsetenv("NAME")
		os.Unsetenv("ADDRESS_CITY")
		os.Unsetenv("ADDRESS_STATE")
	}()

	expected := Person{
		Name: "John Doe",
		Address: Address{
			City:  "New York",
			State: "NY",
		},
	}

	var person Person
	e := New()
	if err := e.ParseEnv(&person); err != nil {
		t.Errorf("Failed to parse nested environment variables: %s", err)
	}

	if !reflect.DeepEqual(person, expected) {
		t.Errorf("Expected %+v, got %+v", expected, person)
	}
}

type CustomTime struct {
	time.Time
}

func (ct *CustomTime) ParseField(value string) error {
	tm, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return err
	}
	ct.Time = tm
	return nil
}

func TestParseEnvCustomType(t *testing.T) {
	type Config struct {
		StartTime CustomTime `enviro:"start_time"`
	}

	startTime := "2023-01-02T15:04:05Z"
	os.Setenv("START_TIME", startTime)
	defer func() {
		os.Unsetenv("START_TIME")
	}()

	var config Config
	e := New()
	if err := e.ParseEnv(&config); err != nil {
		t.Errorf("Failed to parse custom type environment variable: %s", err)
	}

	expectedTime, _ := time.Parse(time.RFC3339, startTime)
	if !config.StartTime.Time.Equal(expectedTime) {
		t.Errorf("Expected %s, got %s", expectedTime, config.StartTime.Time)
	}
}
