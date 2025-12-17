package relay

import (
	"context"
	"crypto/tls"
	"log"
	"log/slog"
	"net/http"
	"sync"

	"github.com/okdaichi/gomoqt/moqt"
	"github.com/okdaichi/gomoqt/quic"
	"github.com/okdaichi/qumo/relay/health"
)

type Server struct {
	Addr       string
	TLSConfig  *tls.Config
	QUICConfig *quic.Config
	Config     *Config

	CheckHTTPOrigin func(r *http.Request) bool

	Client *moqt.Client

	TrackMux *moqt.TrackMux

	clientTrackMux *moqt.TrackMux

	client *moqt.Client
	server *moqt.Server

	// Health check support (optional)
	Health *health.StatusHandler

	initOnce sync.Once
}

func (s *Server) init() {
	s.initOnce.Do(func() {
		if s.Config == nil {
			s.Config = &Config{}
		}

		if s.TLSConfig == nil {
			panic("no tls config")
		}

		if s.TrackMux == nil {
			s.TrackMux = moqt.DefaultMux
		}

		s.clientTrackMux = moqt.NewTrackMux()
	})
}

func (s *Server) ListenAndServe() error {
	s.init()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverMux := s.TrackMux
	clientMux := s.clientTrackMux

	s.server = &moqt.Server{
		Addr:            s.Addr,
		TLSConfig:       s.TLSConfig,
		QUICConfig:      s.QUICConfig,
		CheckHTTPOrigin: s.CheckHTTPOrigin,
		SetupHandler: moqt.SetupHandlerFunc(func(w moqt.SetupResponseWriter, r *moqt.SetupRequest) {
			downstream, err := moqt.Accept(w, r, serverMux)
			if err != nil {
				slog.Error("failed to accept connection", "err", err)
				return
			}

			// Track connection
			s.Health.IncrementConnections()
			defer func() {
				downstream.CloseWithError(moqt.NoError, moqt.SessionErrorText(moqt.NoError))
				s.Health.DecrementConnections()
			}()

			err = Relay(ctx, downstream, func(handler *RelayHandler) {
				// Announce to downstream peers with server mux
				serverMux.Announce(handler.Announcement, handler)
				// Announce to upstream with client mux
				clientMux.Announce(handler.Announcement, handler)
			})

			if err != nil {
				slog.Error("relay session ended", "err", err)
				return
			}
		}),
	}

	var wg sync.WaitGroup

	// Only connect to upstream if URL is provided
	if s.Config.Upstream != "" {
		s.client = &moqt.Client{
			TLSConfig:  s.TLSConfig,
			QUICConfig: s.QUICConfig,
		}

		wg.Go(func() {
			upstream, err := s.client.Dial(ctx, s.Config.Upstream, clientMux)
			if err != nil {
				log.Printf("Failed to connect to upstream: %v", err)
				s.Health.SetUpstreamConnected(false)
				return
			}
			s.Health.SetUpstreamConnected(true)
			log.Printf("Connected to upstream: %s", s.Config.Upstream)

			defer func() {
				upstream.CloseWithError(moqt.NoError, moqt.SessionErrorText(moqt.NoError))
				s.Health.SetUpstreamConnected(false)
			}()

			err = Relay(ctx, upstream, func(handler *RelayHandler) {
				// Announce to downstream peers with server mux
				serverMux.Announce(handler.Announcement, handler)
			})
			if err != nil {
				return
			}

		})
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := s.server.HandleWebTransport(w, r)
		if err != nil {
			slog.Error("failed to handle web transport", "err", err)
		}
	})

	// Start server - this will block until server closes
	err := s.server.ListenAndServe()

	// Wait for upstream goroutine to finish (if it was started)
	wg.Wait()

	return err
}

func (s *Server) Close() error {
	//
	s.init()

	if s.server != nil {
		_ = s.server.Close()
	}

	if s.client != nil {
		_ = s.client.Close()
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	//
	s.init()

	if s.server != nil {
		done := make(chan error, 1)
		go func() {
			done <- s.server.Shutdown(ctx)
		}()

		select {
		case err := <-done:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if s.client != nil {
		done := make(chan error, 1)
		go func() {
			done <- s.client.Shutdown(ctx)
		}()

		select {
		case err := <-done:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
