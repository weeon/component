package app

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/weeon/contract"
	"os"
)

type RegisterFn func(c ServiceConfig) (func(), error)

func ConsulClientFromEnv() (*api.Client, error) {
	consulAddr := os.Getenv("CONSUL_ADDR")
	consulToken := os.Getenv("CONSUL_TOKEN")
	client, err := api.NewClient(&api.Config{
		Address: consulAddr,
		Token:   consulToken,
	})
	return client, err
}

func NewConsulRegister(cli *api.Client, logger contract.Logger) RegisterFn {
	var err error
	return func(c ServiceConfig) (func(), error) {
		id := fmt.Sprintf("%s-%s", c.ServiceKey, c.IP)
		r := api.AgentServiceRegistration{
			ID:      id,
			Name:    c.ServiceKey,
			Tags:    []string{c.Namespace, c.ServiceKey, c.Service},
			Port:    c.HttpPort,
			Address: c.IP,
			Check: &api.AgentServiceCheck{
				Interval:                       "3s",
				Timeout:                        "3s",
				HTTP:                           fmt.Sprintf("http://%s:%d", c.IP, c.HttpPort),
				DeregisterCriticalServiceAfter: "30s",
			},
		}

		err = cli.Agent().ServiceRegister(&r)
		if err != nil {
			return nil, err
		}

		deregisterFn := func() {
			logger.Infof("deregister %s", id)
			err = cli.Agent().ServiceDeregister(id)
			if err != nil {
				logger.Errorw("Service  Deregister error ",
					"service_id", id,
					"error_message", err,
				)
				fmt.Println(err)
			}
		}

		return deregisterFn, err
	}
}
