/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package settings

import (
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/parnurzeal/gorequest"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/ingress-nginx/test/e2e/framework"
)

const (
	waitForLuaSync = 5 * time.Second
)

var _ = framework.IngressNginxDescribe("Disabled catch-all", func() {
	f := framework.NewDefaultFramework("disabled-catch-all")

	BeforeEach(func() {
		f.NewEchoDeploymentWithArgs([]string{"--disable-catch-all=true"})
	})

	AfterEach(func() {
	})

	It("should ignore catch all Ingress", func() {
		host := "foo"

		ing := framework.NewSingleCatchAllIngress("catch-all", f.IngressController.Namespace, "http-svc", 80, nil)
		f.EnsureIngress(ing)

		ing = framework.NewSingleIngress(host, "/", host, f.IngressController.Namespace, "http-svc", 80, nil)
		f.EnsureIngress(ing)

		time.Sleep(waitForLuaSync)

		f.WaitForNginxServer(host, func(cfg string) bool {
			return strings.Contains(cfg, "server_name foo")
		})

		f.WaitForNginxServer("_", func(cfg string) bool {
			return strings.Contains(cfg, `set $ingress_name ""`) &&
				strings.Contains(cfg, `set $proxy_upstream_name "upstream-default-backend"`)
		})
	})

	It("should delete Ingress updated to catch-all", func() {
		host := "foo"

		ing := framework.NewSingleIngress(host, "/", host, f.IngressController.Namespace, "http-svc", 80, nil)
		f.EnsureIngress(ing)

		f.WaitForNginxServer(host,
			func(server string) bool {
				return strings.Contains(server, "server_name foo")
			})

		resp, _, errs := gorequest.New().
			Get(f.IngressController.HTTPURL).
			Set("Host", host).
			End()
		Expect(errs).To(BeNil())
		Expect(resp.StatusCode).Should(Equal(http.StatusOK))

		err := framework.UpdateIngress(f.KubeClientSet, f.IngressController.Namespace, host, func(ingress *extensions.Ingress) error {
			ingress.Spec.Rules = nil
			ingress.Spec.Backend = &extensions.IngressBackend{
				ServiceName: "http-svc",
				ServicePort: intstr.FromInt(80),
			}
			return nil
		})
		Expect(err).ToNot(HaveOccurred())

		time.Sleep(waitForLuaSync)

		f.WaitForNginxConfiguration(func(cfg string) bool {
			return !strings.Contains(cfg, "server_name foo")
		})
	})
})
