package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/edgelesssys/coordinator/coordinator/config"
	"github.com/edgelesssys/coordinator/coordinator/core"
	"github.com/edgelesssys/coordinator/coordinator/quote"
	"github.com/edgelesssys/coordinator/coordinator/server"
	"github.com/edgelesssys/coordinator/util"
)

func run(validator quote.Validator, issuer quote.Issuer, sealKey []byte, sealDirPrefix string) {
	log.SetPrefix("[Coordinator] ")
	log.Println("starting coordinator")

	// fetching env vars
	log.Println("fetching env variables")
	sealDir := util.MustGetenv(config.SealDir)
	sealDir = filepath.Join(sealDirPrefix, sealDir)
	dnsNamesString := util.MustGetenv(config.DNSNames)
	dnsNames := strings.Split(dnsNamesString, ",")
	clientServerAddr := util.MustGetenv(config.ClientAddr)
	meshServerAddr := util.MustGetenv(config.MeshAddr)

	// creating core
	log.Println("creating the Core object")
	if err := os.MkdirAll(sealDir, 0700); err != nil {
		panic(err)
	}
	sealer := core.NewAESGCMSealer(sealDir, sealKey)
	core, err := core.NewCore("Coordinator", dnsNames, validator, issuer, sealer)
	if err != nil {
		panic(err)
	}

	// start client server
	log.Println("starting the client server")
	mux := server.CreateServeMux(core)
	clientServerTLSConfig, err := core.GetTLSConfig()
	if err != nil {
		panic(err)
	}
	go server.RunClientServer(mux, clientServerAddr, clientServerTLSConfig)

	// run marble server
	log.Println("starting the marble server")
	addrChan := make(chan string)
	errChan := make(chan error)
	go server.RunMarbleServer(core, meshServerAddr, addrChan, errChan)
	for {
		select {
		case err := <-errChan:
			if err != nil {
				panic(err)
			}
			return
		case grpcAddr := <-addrChan:
			log.Println("started gRPC server at ", grpcAddr)
		}
	}
}
