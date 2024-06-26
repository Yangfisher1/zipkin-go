// Copyright 2022 The Yangfisher1 Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zipkin_test

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/Yangfisher1/zipkin-go"
	"github.com/Yangfisher1/zipkin-go/model"
)

const (
	serviceName           = "my_service"
	onlyHost              = "localhost"
	defaultPort           = 0
	port                  = 8081
	invalidNegativePort   = "localhost:-8081"
	invalidOutOfRangePort = "localhost:65536"
	invalidHostPort       = "::1:8081"
	unreachableHostPort   = "nosuchhost:8081"
)

var (
	ipv4HostPort    = "127.0.0.1:" + fmt.Sprintf("%d", port)
	ipv6HostPort    = "[2001:db8::68]:" + fmt.Sprintf("%d", port)
	ipv4ForHostPort = net.IPv4(127, 0, 0, 1)
	ipv6ForHostPort = net.ParseIP("2001:db8::68")
)

func TestEmptyEndpoint(t *testing.T) {
	ep, err := zipkin.NewEndpoint("", "")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if ep != nil {
		t.Errorf("endpoint want nil, have: %+v", ep)
	}
}

func TestServiceNameOnlyEndpoint(t *testing.T) {
	have, err := zipkin.NewEndpoint(serviceName, "")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	want := &model.Endpoint{ServiceName: serviceName}
	if !reflect.DeepEqual(want, have) {
		t.Errorf("endpoint want %+v, have: %+v", want, have)
	}
}

func TestInvalidHostPort(t *testing.T) {
	_, err := zipkin.NewEndpoint(serviceName, invalidHostPort)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "too many colons in address") {
		t.Fatalf("expected too many colons in address error, got: %s", err.Error())
	}
}

func TestNewEndpointFailsDueToOutOfRangePort(t *testing.T) {
	_, err := zipkin.NewEndpoint(serviceName, invalidOutOfRangePort)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "value out of range") {
		t.Fatalf("expected out of range error, got: %s", err.Error())
	}
}

func TestNewEndpointFailsDueToNegativePort(t *testing.T) {
	_, err := zipkin.NewEndpoint(serviceName, invalidNegativePort)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "invalid syntax") {
		t.Fatalf("expected invalid syntax error, got: %s", err.Error())
	}
}

func TestNewEndpointFailsDueToLookupIP(t *testing.T) {
	_, err := zipkin.NewEndpoint(serviceName, unreachableHostPort)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "host lookup failure") {
		t.Fatalf("expected no such host error, got: %s", err.Error())
	}
}

func TestNewEndpointDefaultsPortToZeroWhenMissing(t *testing.T) {
	endpoint, err := zipkin.NewEndpoint(serviceName, onlyHost)

	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if endpoint.Port != defaultPort {
		t.Fatalf("expected port %d, got %d", defaultPort, endpoint.Port)
	}
}

func TestNewEndpointIpv4Success(t *testing.T) {
	endpoint, err := zipkin.NewEndpoint(serviceName, ipv4HostPort)

	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if serviceName != endpoint.ServiceName {
		t.Fatalf("expected service name %s, got %s", serviceName, endpoint.ServiceName)
	}

	if !ipv4ForHostPort.Equal(endpoint.IPv4) {
		t.Fatalf("expected IPv4 %s, got %s", ipv4ForHostPort.String(), endpoint.IPv4.String())
	}

	if port != endpoint.Port {
		t.Fatalf("expected port %d, got %d", port, endpoint.Port)
	}

	if endpoint.IPv6 != nil {
		t.Fatalf("expected empty IPv6 got %+v", endpoint.IPv6)
	}
}

func TestNewEndpointIpv6Success(t *testing.T) {
	endpoint, err := zipkin.NewEndpoint(serviceName, ipv6HostPort)

	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if serviceName != endpoint.ServiceName {
		t.Fatalf("expected service name %s, got %s", serviceName, endpoint.ServiceName)
	}

	if !ipv6ForHostPort.Equal(endpoint.IPv6) {
		t.Fatalf("expected IPv6 %s, got %s", ipv6ForHostPort.String(), endpoint.IPv6.String())
	}

	if port != endpoint.Port {
		t.Fatalf("expected port %d, got %d", port, endpoint.Port)
	}

	if endpoint.IPv4 != nil {
		t.Fatalf("expected empty IPv4 got %+v", endpoint.IPv4)
	}
}
