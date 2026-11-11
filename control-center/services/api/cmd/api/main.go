package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"hcc.local/api/internal/provider"
	"hcc.local/api/internal/server"
	"hcc.local/api/internal/store"
)

func main() {
	addr := env("HCC_HTTP_ADDR", "127.0.0.1:8080")
	if !strings.HasPrefix(addr, "127.0.0.1:") && !strings.HasPrefix(addr, "localhost:") && os.Getenv("HCC_ALLOW_CONTAINER_BIND") != "1" {
		log.Fatal(`{"msg":"refusing non-loopback bind"}`)
	}
	st, err := store.Open(env("HCC_DB_PATH", "data/hcc.sqlite"))
	if err != nil {
		log.Fatalf(`{"msg":"open db","err":%q}`, err.Error())
	}
	defer st.Close()
	if err := st.Migrate(); err != nil {
		log.Fatalf(`{"msg":"migrate","err":%q}`, err.Error())
	}
	var cf provider.CloudflareProvider = provider.Unconfigured{}
	token := os.Getenv("HCC_CF_API_TOKEN")
	account := os.Getenv("HCC_CF_ACCOUNT_ID")
	zone := os.Getenv("HCC_CF_ZONE_ID")
	if token != "" && account != "" && zone != "" {
		cf = provider.NewHTTP(token, account, zone, os.Getenv("HCC_CF_TUNNEL_ID"))
		log.Printf(`{"msg":"cloudflare read-only enabled"}`)
	} else {
		log.Printf(`{"msg":"cloudflare management token absent","mode":"read-only-unconfigured"}`)
	}
	if os.Getenv("HCC_ADMIN_PASSWORD") == "" {
		log.Printf(`{"msg":"admin password is empty","login":"disabled"}`)
	}
	if os.Getenv("HCC_AGENT_SECRET") == "" {
		log.Printf(`{"msg":"agent secret is empty","agent":"denied"}`)
	}
	srv := &server.Server{
		Store:    st,
		Password: os.Getenv("HCC_ADMIN_PASSWORD"),
		Secret:   os.Getenv("HCC_AGENT_SECRET"),
		Docker:   provider.ObservedDocker{},
		CF:       cf,
	}
	log.Printf(`{"msg":"listen","addr":%q}`, addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
