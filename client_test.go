package gortsplib

import (
	"crypto/tls"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aler9/gortsplib/pkg/auth"
	"github.com/aler9/gortsplib/pkg/base"
	"github.com/aler9/gortsplib/pkg/conn"
	"github.com/aler9/gortsplib/pkg/url"
)

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestClientTLSSetServerName(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:8554")
	require.NoError(t, err)
	defer l.Close()

	serverDone := make(chan struct{})
	defer func() { <-serverDone }()
	go func() {
		defer close(serverDone)

		nconn, err := l.Accept()
		require.NoError(t, err)
		defer nconn.Close()

		cert, err := tls.X509KeyPair(serverCert, serverKey)
		require.NoError(t, err)

		tnconn := tls.Server(nconn, &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
			VerifyConnection: func(cs tls.ConnectionState) error {
				require.Equal(t, "localhost", cs.ServerName)
				return nil
			},
		})

		err = tnconn.Handshake()
		require.EqualError(t, err, "remote error: tls: bad certificate")
	}()

	u, err := url.Parse("rtsps://localhost:8554/stream")
	require.NoError(t, err)

	c := Client{
		TLSConfig: &tls.Config{},
	}

	err = c.Start(u.Scheme, u.Host)
	require.NoError(t, err)
	defer c.Close()

	_, err = c.Options(u)
	require.Error(t, err)

	<-serverDone
}

func TestClientSession(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:8554")
	require.NoError(t, err)
	defer l.Close()

	serverDone := make(chan struct{})
	defer func() { <-serverDone }()
	go func() {
		defer close(serverDone)

		nconn, err := l.Accept()
		require.NoError(t, err)
		conn := conn.NewConn(nconn)
		defer nconn.Close()

		req, err := conn.ReadRequest()
		require.NoError(t, err)
		require.Equal(t, base.Options, req.Method)

		err = conn.WriteResponse(&base.Response{
			StatusCode: base.StatusOK,
			Header: base.Header{
				"Public": base.HeaderValue{strings.Join([]string{
					string(base.Describe),
				}, ", ")},
				"Session": base.HeaderValue{"123456"},
			},
		})
		require.NoError(t, err)

		req, err = conn.ReadRequest()
		require.NoError(t, err)
		require.Equal(t, base.Describe, req.Method)

		require.Equal(t, base.HeaderValue{"123456"}, req.Header["Session"])

		track := &TrackH264{
			PayloadType:       96,
			SPS:               []byte{0x01, 0x02, 0x03, 0x04},
			PPS:               []byte{0x01, 0x02, 0x03, 0x04},
			PacketizationMode: 1,
		}

		tracks := Tracks{track}
		tracks.setControls()

		err = conn.WriteResponse(&base.Response{
			StatusCode: base.StatusOK,
			Header: base.Header{
				"Content-Type": base.HeaderValue{"application/sdp"},
				"Session":      base.HeaderValue{"123456"},
			},
			Body: tracks.Marshal(false),
		})
		require.NoError(t, err)
	}()

	u, err := url.Parse("rtsp://localhost:8554/stream")
	require.NoError(t, err)

	c := Client{}

	err = c.Start(u.Scheme, u.Host)
	require.NoError(t, err)
	defer c.Close()

	_, _, _, err = c.Describe(u)
	require.NoError(t, err)
}

func TestClientAuth(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:8554")
	require.NoError(t, err)
	defer l.Close()

	serverDone := make(chan struct{})
	defer func() { <-serverDone }()
	go func() {
		defer close(serverDone)

		nconn, err := l.Accept()
		require.NoError(t, err)
		conn := conn.NewConn(nconn)
		defer nconn.Close()

		req, err := conn.ReadRequest()
		require.NoError(t, err)
		require.Equal(t, base.Options, req.Method)

		err = conn.WriteResponse(&base.Response{
			StatusCode: base.StatusOK,
			Header: base.Header{
				"Public": base.HeaderValue{strings.Join([]string{
					string(base.Describe),
				}, ", ")},
			},
		})
		require.NoError(t, err)

		req, err = conn.ReadRequest()
		require.NoError(t, err)
		require.Equal(t, base.Describe, req.Method)

		v := auth.NewValidator("myuser", "mypass", nil)

		err = conn.WriteResponse(&base.Response{
			StatusCode: base.StatusUnauthorized,
			Header: base.Header{
				"WWW-Authenticate": v.Header(),
			},
		})
		require.NoError(t, err)

		req, err = conn.ReadRequest()
		require.NoError(t, err)
		require.Equal(t, base.Describe, req.Method)

		err = v.ValidateRequest(req)
		require.NoError(t, err)

		track := &TrackH264{
			PayloadType:       96,
			SPS:               []byte{0x01, 0x02, 0x03, 0x04},
			PPS:               []byte{0x01, 0x02, 0x03, 0x04},
			PacketizationMode: 1,
		}

		tracks := Tracks{track}
		tracks.setControls()

		err = conn.WriteResponse(&base.Response{
			StatusCode: base.StatusOK,
			Header: base.Header{
				"Content-Type": base.HeaderValue{"application/sdp"},
			},
			Body: tracks.Marshal(false),
		})
		require.NoError(t, err)
	}()

	u, err := url.Parse("rtsp://myuser:mypass@localhost:8554/stream")
	require.NoError(t, err)

	c := Client{}

	err = c.Start(u.Scheme, u.Host)
	require.NoError(t, err)
	defer c.Close()

	_, _, _, err = c.Describe(u)
	require.NoError(t, err)
}

func TestClientDescribeCharset(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:8554")
	require.NoError(t, err)
	defer l.Close()

	serverDone := make(chan struct{})
	defer func() { <-serverDone }()
	go func() {
		defer close(serverDone)

		nconn, err := l.Accept()
		require.NoError(t, err)
		defer nconn.Close()
		conn := conn.NewConn(nconn)

		req, err := conn.ReadRequest()
		require.NoError(t, err)
		require.Equal(t, base.Options, req.Method)

		err = conn.WriteResponse(&base.Response{
			StatusCode: base.StatusOK,
			Header: base.Header{
				"Public": base.HeaderValue{strings.Join([]string{
					string(base.Describe),
				}, ", ")},
			},
		})
		require.NoError(t, err)

		req, err = conn.ReadRequest()
		require.NoError(t, err)
		require.Equal(t, base.Describe, req.Method)
		require.Equal(t, mustParseURL("rtsp://localhost:8554/teststream"), req.URL)

		track1 := &TrackH264{
			PayloadType:       96,
			SPS:               []byte{0x01, 0x02, 0x03, 0x04},
			PPS:               []byte{0x01, 0x02, 0x03, 0x04},
			PacketizationMode: 1,
		}

		err = conn.WriteResponse(&base.Response{
			StatusCode: base.StatusOK,
			Header: base.Header{
				"Content-Type": base.HeaderValue{"application/sdp; charset=utf-8"},
				"Content-Base": base.HeaderValue{"rtsp://localhost:8554/teststream/"},
			},
			Body: Tracks{track1}.Marshal(false),
		})
		require.NoError(t, err)
	}()

	u, err := url.Parse("rtsp://localhost:8554/teststream")
	require.NoError(t, err)

	c := Client{}

	err = c.Start(u.Scheme, u.Host)
	require.NoError(t, err)
	defer c.Close()

	_, _, _, err = c.Describe(u)
	require.NoError(t, err)
}

func TestClientClose(t *testing.T) {
	u, err := url.Parse("rtsp://localhost:8554/teststream")
	require.NoError(t, err)

	c := Client{}

	err = c.Start(u.Scheme, u.Host)
	require.NoError(t, err)

	c.Close()

	_, err = c.Options(u)
	require.EqualError(t, err, "terminated")

	_, _, _, err = c.Describe(u)
	require.EqualError(t, err, "terminated")

	_, err = c.Announce(u, nil)
	require.EqualError(t, err, "terminated")

	_, err = c.Setup(nil, nil, 0, 0)
	require.EqualError(t, err, "terminated")

	_, err = c.Play(nil)
	require.EqualError(t, err, "terminated")

	_, err = c.Record()
	require.EqualError(t, err, "terminated")

	_, err = c.Pause()
	require.EqualError(t, err, "terminated")
}

func TestClientCloseDuringRequest(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:8554")
	require.NoError(t, err)
	defer l.Close()

	requestReceived := make(chan struct{})
	releaseConn := make(chan struct{})

	serverDone := make(chan struct{})
	defer func() { <-serverDone }()
	go func() {
		defer close(serverDone)

		nconn, err := l.Accept()
		require.NoError(t, err)
		defer nconn.Close()
		conn := conn.NewConn(nconn)

		req, err := conn.ReadRequest()
		require.NoError(t, err)
		require.Equal(t, base.Options, req.Method)

		close(requestReceived)
		<-releaseConn
	}()

	u, err := url.Parse("rtsp://localhost:8554/teststream")
	require.NoError(t, err)

	c := Client{}

	err = c.Start(u.Scheme, u.Host)
	require.NoError(t, err)

	optionsDone := make(chan struct{})
	go func() {
		defer close(optionsDone)
		_, err := c.Options(u)
		require.Error(t, err)
	}()

	<-requestReceived
	c.Close()
	<-optionsDone
	close(releaseConn)
}
