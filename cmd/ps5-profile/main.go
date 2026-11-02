// Command ps5-profile serves the PSN user-profile endpoints.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openpak/ps5-common/accountcore"
	"github.com/openpak/ps5-common/config"
	"github.com/openpak/ps5-common/httpx"
	"github.com/openpak/ps5-common/oauth"

	"openpak/ps5-profile/internal/profile"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(2)
	}
	signer, err := oauth.NewSigner([]byte(cfg.TokenKey))
	if err != nil {
		log.Error("PS5_TOKEN_KEY", "err", err)
		os.Exit(2)
	}
	var core accountcore.Core
	if cfg.CoreURL != "" {
		if core, err = accountcore.New(cfg.CoreURL, cfg.CoreToken, nil); err != nil {
			log.Error("account core", "err", err)
			os.Exit(2)
		}
	} else {
		log.Warn("PS5_ACCOUNT_CORE_URL unset: other accounts' profiles are not served (in-memory dev mode)")
		core = accountcore.NewMemory()
	}
	s := profile.New(core, signer, os.Getenv("PS5_PROFILE_AVATAR_URL"))
	srv := &http.Server{Addr: cfg.Addr, Handler: httpx.Logger(log, s.Handler()), ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()
	log.Info("ps5-profile listening", "addr", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}
