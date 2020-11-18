package app

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/weeon/mod"
	"testing"
)

type write struct {
	b []byte
}

func (w write) Write(p []byte) (n int, err error) {
	fmt.Println(string(p))
	w.b = append(w.b, p...)
	return len(p), nil
}

func TestConf(t *testing.T) {
	conf := NewConfig()
	conf.Database["main"] = mod.Database{
		Host: "111",
	}
	w := write{}
	err := toml.NewEncoder(w).Encode(conf.Database)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(w.b))
}
