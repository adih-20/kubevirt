/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package standalone_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher/env-config"

	"kubevirt.io/kubevirt/pkg/virt-launcher/standalone"
	virtwrap "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
)

var _ = Describe("HandleStandaloneMode", func() {
	var (
		mockCtrl *gomock.Controller
		mockDM   *virtwrap.MockDomainManager
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDM = virtwrap.NewMockDomainManager(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should do nothing if STANDALONE_VMI env var is not set", func() {
		os.Unsetenv("STANDALONE_VMI")
		config := virtlauncher.ReadVirtLauncherConfig()
		standalone.HandleStandaloneMode(mockDM, config)
	})

	It("should panic on invalid JSON in STANDALONE_VMI", func() {
		os.Setenv("STANDALONE_VMI", "invalid json")
		config := virtlauncher.ReadVirtLauncherConfig()
		defer os.Unsetenv("STANDALONE_VMI")

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM, config)
		}).To(Panic())
	})

	It("should panic if SyncVMI fails", func() {
		vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance","metadata":{"name":"testvmi"}}`
		os.Setenv("STANDALONE_VMI", vmiJSON)
		config := virtlauncher.ReadVirtLauncherConfig()
		defer os.Unsetenv("STANDALONE_VMI")

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(nil, fmt.Errorf("sync error"))

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM, config)
		}).To(PanicWith(MatchError(ContainSubstring("sync error"))))
	})

	It("should succeed with valid JSON and successful SyncVMI", func() {
		vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance","metadata":{"name":"testvmi"}}`
		os.Setenv("STANDALONE_VMI", vmiJSON)
		config := virtlauncher.ReadVirtLauncherConfig()
		defer os.Unsetenv("STANDALONE_VMI")

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(nil, nil)

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM, config)
		}).NotTo(Panic())
	})

	It("should succeed with valid YAML and successful SyncVMI", func() {
		vmiYAML := `apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: testvmi-yaml`
		os.Setenv("STANDALONE_VMI", vmiYAML)
		config := virtlauncher.ReadVirtLauncherConfig()
		defer os.Unsetenv("STANDALONE_VMI")

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(nil, nil)

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM, config)
		}).NotTo(Panic())
	})

	It("should panic on invalid YAML in STANDALONE_VMI", func() {
		os.Setenv("STANDALONE_VMI", "invalid: yaml: here")
		config := virtlauncher.ReadVirtLauncherConfig()
		defer os.Unsetenv("STANDALONE_VMI")

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM, config)
		}).To(Panic())
	})
})
