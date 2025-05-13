package framework

import "flag"

type TestContextType struct {
	ReportPrefix string
	ReportDir    string
}

var TestContext TestContextType

func RegisterCommonFlags(flags *flag.FlagSet) {
	flags.StringVar(&TestContext.ReportPrefix, "report-prefix", "", "Optional prefix for JUnit XML reports. Default is empty, which doesn't prepend anything to the default name.")
	flags.StringVar(&TestContext.ReportDir, "report-dir", "", "Path to the directory where the JUnit XML reports should be saved. Default is empty, which doesn't generate these reports.")
}
