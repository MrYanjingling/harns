package e2e

import (
	"flag"
	"fmt"
	"lightiot/pkg/version"
	"lightiot/test/framework"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"lightiot/test/e2e/common"
	_ "lightiot/test/e2e/control"
	_ "lightiot/test/e2e/event"
	_ "lightiot/test/e2e/model"
	_ "lightiot/test/e2e/notification"
	_ "lightiot/test/e2e/rule"
)

func handleFlags() {
	framework.RegisterCommonFlags(flag.CommandLine)
	common.Flags(flag.CommandLine)
	flag.Parse()
}

func TestMain(m *testing.M) {
	var versionFlag bool
	flag.CommandLine.BoolVar(&versionFlag, "version", false, "Displays version information.")
	handleFlags()

	if versionFlag {
		fmt.Printf("%s\n", version.Get())
		os.Exit(0)
	}

	rand.Seed(time.Now().UnixNano())
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	logs.InitLogs()
	defer logs.FlushLogs()

	gomega.RegisterFailHandler(ginkgo.Fail)

	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()

	// TODO: https://github.com/onsi/ginkgo/blob/ver2/docs/MIGRATING_TO_V2.md#spec-labels
	// Disable skipped tests unless they are explicitly requested.
	if len(suiteConfig.FocusStrings) == 0 && len(suiteConfig.SkipStrings) == 0 {
		suiteConfig.SkipStrings = []string{`\[Flaky\]|\[Feature:.+\]`}
	}

	// TODO: https://github.com/onsi/ginkgo/blob/ver2/docs/MIGRATING_TO_V2.md#removed-custom-reporters
	// Run tests through the Ginkgo runner with output to console + JUnit for Jenkins
	// var r []ginkgo.Reporter
	// if framework.TestContext.ReportDir != "" {
	// 	// TODO: we should probably only be trying to create this directory once
	// 	// rather than once-per-Ginkgo-node.
	// 	if err := os.MkdirAll(framework.TestContext.ReportDir, 0755); err != nil {
	// 		klog.ErrorS(err, "Failed creating report directory")
	// 	} else {
	// 		r = append(r, reporters.NewJUnitReporter(path.Join(framework.TestContext.ReportDir, fmt.Sprintf("junit_%v%02d.xml", framework.TestContext.ReportPrefix, ginkgo.GinkgoParallelProcess()))))
	// 	}
	// }
	klog.InfoS("Started e2e run", "id", uuid.New(), "GinkgoProcess", ginkgo.GinkgoParallelProcess())
	ginkgo.RunSpecs(t, "IoT e2e suite", suiteConfig, reporterConfig)
}
