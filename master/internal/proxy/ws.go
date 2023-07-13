package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"math"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func connectWithbackoff(tlsConn *tls.Conn) error {
	var err error
	for x := 0; x < 3; x++ {
		err = tlsConn.Handshake()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(math.Pow(2, float64(x))) * time.Second)
	}
	return err
}

func newSingleHostReverseWebSocketProxy(c echo.Context, t *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		in, _, err := c.Response().Hijack()
		if err != nil {
			c.Error(errors.Errorf("error hijacking connection to %v: %v", t, err))
			return
		}
		defer func() {
			if cerr := in.Close(); cerr != nil {
				c.Logger().Error(cerr)
			}
		}()

		out, err := net.Dial("tcp", t.Host)
		if err != nil {
			c.Error(echo.NewHTTPError(http.StatusBadGateway,
				errors.Errorf("error dialing to %v: %v", t, err)))
			return
		}
		defer func() {
			if cerr := out.Close(); cerr != nil {
				c.Logger().Error(cerr)
			}
		}()

		if t.Scheme == https {
			keyBytes, certBytes, err := MasterKeyAndCert()
			if err != nil {
				c.Error(echo.NewHTTPError(http.StatusBadGateway,
					errors.Errorf("error getting tls key or cert: %v", err)))
			}
			cert, err := tls.X509KeyPair(certBytes, keyBytes)
			if err != nil {
				return
			}

			masterCaBytes, err := MasterCACert()
			if err != nil {
				return
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(masterCaBytes)

			//nolint:gosec,G402
			tlsConfig := &tls.Config{
				ServerName:         "127.0.0.1",
				Certificates:       []tls.Certificate{cert},
				RootCAs:            caCertPool,
				InsecureSkipVerify: true,
				VerifyConnection:   VerifyMasterSigned,
			}
			tlsConn := tls.Client(out, tlsConfig)
			out = tlsConn

			err = connectWithbackoff(tlsConn)
			if err != nil {
				c.Error(echo.NewHTTPError(http.StatusBadGateway,
					errors.Errorf("tls handshake error %v", err)))
			}
		}

		err = r.Write(out)
		if err != nil {
			c.Error(echo.NewHTTPError(http.StatusBadGateway,
				errors.Errorf("error copying headers for %v: %v", t, err)))
			return
		}

		copyReqErr := asyncCopy(out, in)
		copyResErr := asyncCopy(in, out)
		if cerr := <-copyReqErr; cerr != nil {
			c.Logger().Errorf("error copying request body for %v: %v", t, cerr)
		}
		if cerr := <-copyResErr; cerr != nil {
			c.Logger().Errorf("error copying response body for %v: %v", t, cerr)
		}
	})
}
