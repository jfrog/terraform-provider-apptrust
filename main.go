package main

import (
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	provider "github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/provider"
)

// Run the docs generation tool, check its repository for more information on how it works and how docs
// can be customized.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	var serveOpts []tf6server.ServeOpt

	if debug {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	err := tf6server.Serve(
		"registry.terraform.io/jfrog/apptrust",
		providerserver.NewProtocol6(provider.Framework()()),
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err)
	}
}
