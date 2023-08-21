package node

import (
	"Satogram/storage"
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/lightninglabs/lndclient"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lnrpc/walletrpc"
	"github.com/lightningnetwork/lnd/lnrpc/wtclientrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type LND struct {
	client lnrpc.LightningClient
	state  lnrpc.StateClient
	router routerrpc.RouterClient
	wt     wtclientrpc.WatchtowerClientClient
	wallet walletrpc.WalletKitClient
	logger *logrus.Logger
	cfg    *LNDConfig
	store  *storage.Bolt
}

type LNDConfig struct {
	Host         string `long:"host" description:"Endpoint for LND"`
	MacaroonPath string `long:"macaroon-path" description:"Path of the macaroon file"`
	TLSPath      string `long:"tls-cert-path" description:"Path of the TLS certificate"`
	Network      string `long:"network" description:"The bitcoin network. Changing this won't modify the paths of other flags."`
}

var conn = &grpc.ClientConn{}

func NewLND(store *storage.Bolt, host, tlsPath, macaroonPath, network string) (LND, error) {
	return LND{
		cfg: &LNDConfig{
			Host:         host,
			TLSPath:      tlsPath,
			MacaroonPath: macaroonPath,
			Network:      network,
		},
		store: store,
	}, nil
}

func (lnd *LND) Ping(ctx context.Context) error {
	// ping test we can hit the node without error
	_, err := lnd.client.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return fmt.Errorf("node unreachable with error: %w", err)

	}
	return nil
}

func (lnd *LND) EstablishClientConnections(ctx context.Context) error {
	dir, file := path.Split(lnd.cfg.MacaroonPath)
	if _, err := os.Stat(lnd.cfg.MacaroonPath); err != nil {
		if err := conn.Close(); err != nil {
			lnd.logger.WithError(err).Error("error closing cancelLndConn connection")
			return err
		}
		return fmt.Errorf("macaroon path error: %w given macaroon path: %s", errors.Unwrap(err), lnd.cfg.MacaroonPath) // unwrap should return a syscall.Errno
	}
	hasTLSData := false
	tlsFileInfo, err := os.Stat(lnd.cfg.TLSPath)
	if err != nil {
		hasTLSData = false
	} else {
		hasTLSData = tlsFileInfo.Size() > 0
	}

	servicesConfig := &lndclient.LndServicesConfig{
		LndAddress:         lnd.cfg.Host,
		Network:            lndclient.Network(lnd.cfg.Network),
		CustomMacaroonPath: lnd.cfg.MacaroonPath,
		SystemCert:         false,
		TLSPath:            lnd.cfg.TLSPath,
	}

	options := []lndclient.BasicClientOption{lndclient.MacFilename(file)}
	if !hasTLSData {
		options = append(options, lndclient.SystemCerts())
		servicesConfig.SystemCert = true
		servicesConfig.TLSPath = ""
	}

	conn, err := lndclient.NewBasicConn(servicesConfig.LndAddress, servicesConfig.TLSPath, dir, lnd.cfg.Network, options...)
	if err != nil {
		return fmt.Errorf("error making connection to LND: %w", err)
	}

	lnd.client = lnrpc.NewLightningClient(conn)
	lnd.state = lnrpc.NewStateClient(conn)
	lnd.router = routerrpc.NewRouterClient(conn)
	lnd.wt = wtclientrpc.NewWatchtowerClientClient(conn)
	lnd.wallet = walletrpc.NewWalletKitClient(conn)
	go func() {
		<-ctx.Done()
		lnd.logger.Debug("context cancelled, closing grpc connection with LND")
		// does this call block?
		conn.Close()
	}()

	return nil
}
func (lnd *LND) GetNodes(ctx context.Context) (map[string]string, error) {
	pubkeys := map[string]string{}
	graph, err := lnd.client.DescribeGraph(ctx, &lnrpc.ChannelGraphRequest{
		IncludeUnannounced: true,
	})
	if err != nil {
		return pubkeys, fmt.Errorf("error describing graph: %w", err)
	}
	for _, node := range graph.Nodes {
		pubkeys[node.PubKey] = node.Alias
	}
	return pubkeys, nil
}
