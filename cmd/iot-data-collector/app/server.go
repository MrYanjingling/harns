package app

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	_ "k8s.io/component-base/logs/json/register"
	"k8s.io/klog/v2"
	"lightiot/cmd/iot-data-collector/options"
	"lightiot/pkg/data"
	"lightiot/pkg/data/storage"
	"lightiot/pkg/generic"
	baseoptions "lightiot/pkg/generic/options"
	"lightiot/pkg/version"
	"lightiot/pkg/version/verflag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const (
	ComponentDataCollector = "iot-data-collector"
)

func NewDataCollectorCmd() *cobra.Command {
	cleanFlagSet := pflag.NewFlagSet(ComponentDataCollector, pflag.ContinueOnError)
	o := options.NewDefaultOptions()
	cmd := &cobra.Command{
		Use: ComponentDataCollector,
		Long: `The IoT data collector serves as a http server that receives
time series data from external or IoT data broker, then persists the data.`,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// initial flag parse, since we disable cobra's flag parsing
			if err := cleanFlagSet.Parse(args); err != nil {
				klog.ErrorS(err, "Failed to parse flag")
				_ = cmd.Usage()
				os.Exit(1)
			}

			// check if there are non-flag arguments in the command line
			cmds := cleanFlagSet.Args()
			if len(cmds) > 0 {
				klog.ErrorS(nil, "Unknown command", "command", cmds[0])
				_ = cmd.Usage()
				os.Exit(1)
			}

			// short-circuit on help
			baseoptions.PrintHelpAndExitIfRequested(cmd, cleanFlagSet)

			// short-circuit on defaultconfig
			baseoptions.PrintDefaultConfigAndExitIfRequested(options.NewDefaultOptions(), cleanFlagSet)

			// short-circuit on verflag
			verflag.PrintAndExitIfRequested()

			if err := baseoptions.ParseAndApplyConfigFile(o, args); err != nil {
				return err
			}

			if errs := options.Validate(o); len(errs) != 0 {
				return utilerrors.NewAggregate(errs)
			}

			// To help debugging, immediately log version
			klog.Infof("Version: %+v", version.Get())
			return run(o)
		},
	}

	verflag.AddFlags(cleanFlagSet)
	o.AddFlags(cleanFlagSet)
	o.AddBaseFlags(cmd, cleanFlagSet)

	return cmd
}

func run(o *options.Options) error {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stopCh := make(chan struct{})
	c, err := o.Config(stopCh)
	if err != nil {
		return err
	}

	exit := startServer(generic.Default(), c.Store, o)

	klog.V(1).InfoS("Server started", "port", o.Port)

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	klog.V(1).InfoS("Shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has some time to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), o.Wait)
	defer cancel()

	exit(ctx)
	close(stopCh)

	return nil
}

func startServer(router *gin.Engine, s *storage.Store, o *options.Options) func(context.Context) {
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowHeaders:     []string{"Content-Type", "Content-Length"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowMethods:     []string{http.MethodPut, http.MethodDelete},
		AllowCredentials: true,
	}))

	data.InstallCollectHandlers(router, s)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", o.Port),
		Handler: router,
	}

	go func() {
		klog.Error(srv.ListenAndServe())
	}()

	return func(ctx context.Context) {
		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			klog.Error(err)
		}
		s.Close()
	}
}
