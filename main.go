package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/jo7oem/hatsukari/internal/bootstrap"
)

func run(args []string, getenv func(string) string, out io.Writer, errOut io.Writer) error {
	conf, err := parseRuntimeConfigWithIO(args, getenv, out, errOut)
	if err != nil {
		if err == errCommandHandled {
			return nil
		}
		return err
	}
	if conf.printConfigExample {
		_, _ = fmt.Fprint(out, runtimeConfigExampleYAML())
		return nil
	}
	return bootstrap.Start(context.Background(), out, bootstrap.RuntimeConfig{
		SiteDir:      conf.siteDir,
		Addr:         conf.addr,
		OTELEnabled:  conf.otelEnabled,
		OTELEndpoint: conf.otelEndpoint,
		OTELInsecure: conf.otelInsecure,
	}, bootstrap.Dependencies{})
}

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}
