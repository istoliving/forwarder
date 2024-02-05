// Copyright 2023 Sauce Labs Inc., all rights reserved.
//
// Copyright 2015 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package martian

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/saucelabs/forwarder/dialvia"
	"github.com/saucelabs/forwarder/internal/martian/log"
	"github.com/saucelabs/forwarder/internal/martian/proxyutil"
	"golang.org/x/net/http/httpguts"
)

// ErrConnectFallback is returned by a ConnectFunc to indicate
// that the CONNECT request should be handled by martian.
var ErrConnectFallback = errors.New("martian: connect fallback")

// ConnectFunc dials a network connection for a CONNECT request.
// If the returned net.Conn is not nil, the response must be not nil.
type ConnectFunc func(req *http.Request) (*http.Response, io.ReadWriteCloser, error)

func (p *Proxy) connect(req *http.Request) (*http.Response, net.Conn, error) {
	var proxyURL *url.URL
	if p.proxyURL != nil {
		u, err := p.proxyURL(req)
		if err != nil {
			return nil, nil, err
		}
		proxyURL = u
	}

	if proxyURL == nil {
		log.Debugf(req.Context(), "CONNECT to host directly: %s", req.URL.Host)

		conn, err := p.dial(req.Context(), "tcp", req.URL.Host)
		if err != nil {
			return nil, nil, err
		}

		return proxyutil.NewResponse(200, http.NoBody, req), conn, nil
	}

	switch proxyURL.Scheme {
	case "http", "https":
		return p.connectHTTP(req, proxyURL)
	case "socks5":
		return p.connectSOCKS5(req, proxyURL)
	default:
		return nil, nil, fmt.Errorf("unsupported proxy scheme: %s", proxyURL.Scheme)
	}
}

func (p *Proxy) connectHTTP(req *http.Request, proxyURL *url.URL) (res *http.Response, conn net.Conn, err error) {
	log.Debugf(req.Context(), "CONNECT with upstream HTTP proxy: %s", proxyURL.Host)

	var d *dialvia.HTTPProxyDialer
	if proxyURL.Scheme == "https" {
		d = dialvia.HTTPSProxy(p.dial, proxyURL, p.clientTLSConfig())
	} else {
		d = dialvia.HTTPProxy(p.dial, proxyURL)
	}
	d.ConnectRequestModifier = p.ConnectRequestModifier

	var ctx context.Context
	if p.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(req.Context(), p.ConnectTimeout)
		defer cancel()
	} else {
		ctx = req.Context()
	}
	res, conn, err = d.DialContextR(ctx, "tcp", req.URL.Host)

	if res != nil {
		if res.StatusCode/100 == 2 {
			res.Body.Close()
			return proxyutil.NewResponse(200, http.NoBody, req), conn, nil
		}

		// If the proxy returns a non-2xx response, return it to the client.
		// But first, replace the Request with the original request.
		res.Request = req
	}

	return res, conn, err
}

func (p *Proxy) clientTLSConfig() *tls.Config {
	if tr, ok := p.roundTripper.(*http.Transport); ok && tr.TLSClientConfig != nil {
		return tr.TLSClientConfig.Clone()
	}

	return &tls.Config{}
}

func (p *Proxy) connectSOCKS5(req *http.Request, proxyURL *url.URL) (*http.Response, net.Conn, error) {
	log.Debugf(req.Context(), "CONNECT with upstream SOCKS5 proxy: %s", proxyURL.Host)

	d := dialvia.SOCKS5Proxy(p.dial, proxyURL)

	conn, err := d.DialContext(req.Context(), "tcp", req.URL.Host)
	if err != nil {
		return nil, nil, err
	}

	return proxyutil.NewResponse(200, http.NoBody, req), conn, nil
}

func upgradeType(h http.Header) string {
	if !httpguts.HeaderValuesContainsToken(h["Connection"], "Upgrade") {
		return ""
	}
	return h.Get("Upgrade")
}
