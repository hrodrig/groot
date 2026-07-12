package notifier

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hrodrig/groot/internal/collector"
	"github.com/hrodrig/groot/internal/config"
)

func TestBuildEmailMessage_format(t *testing.T) {
	msg := buildEmailMessage("from@example.com", []string{"a@x.com", "b@y.com"}, "hello\nworld")
	s := string(msg)
	for _, want := range []string{
		"From: from@example.com",
		"To: a@x.com, b@y.com",
		"Subject: GROOT collect notification",
		"hello",
		"world",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
}

func TestSplitList_semicolonSeparated(t *testing.T) {
	got := splitList(" a@x.com ; ; b@y.com ")
	if len(got) != 2 || got[0] != "a@x.com" || got[1] != "b@y.com" {
		t.Fatalf("got %#v", got)
	}
}

func TestEmailAuth_nilWithoutUsername(t *testing.T) {
	if emailAuth("smtp.example.com", "  ", "secret") != nil {
		t.Fatal("expected nil auth without username")
	}
}

func TestEmailSender_Send_missingHostOrFrom(t *testing.T) {
	e := &emailSender{from: "a@b.com"}
	if err := e.Send(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("err=%v", err)
	}
	e = &emailSender{host: "smtp.example.com"}
	if err := e.Send(context.Background(), "x"); err == nil {
		t.Fatal("expected error")
	}
}

func TestEmailSender_Send_plainSMTP(t *testing.T) {
	srv := startFakeSMTP(t, fakeSMTPOptions{})
	defer srv.Close()

	e := &emailSender{
		host:       "127.0.0.1",
		port:       srv.port,
		from:       "groot@example.com",
		recipients: []string{"ops@example.com"},
	}
	if err := e.Send(context.Background(), "plain body"); err != nil {
		t.Fatal(err)
	}
	if len(srv.messages()) != 1 || !strings.Contains(string(srv.messages()[0]), "plain body") {
		t.Fatalf("messages=%v", srv.messages())
	}
}

func TestEmailSender_Send_STARTTLS(t *testing.T) {
	cert := testTLSCertificate(t)
	srv := startFakeSMTP(t, fakeSMTPOptions{startTLS: true, tlsCert: cert})
	defer srv.Close()

	e := &emailSender{
		host:       "127.0.0.1",
		from:       "groot@example.com",
		recipients: []string{"ops@example.com"},
		skipVerify: true,
	}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(srv.port))
	msg := buildEmailMessage(e.from, e.recipients, "starttls body")
	if err := e.sendSTARTTLS(addr, nil, msg); err != nil {
		t.Fatal(err)
	}
	if len(srv.messages()) != 1 || !strings.Contains(string(srv.messages()[0]), "starttls body") {
		t.Fatalf("messages=%v", srv.messages())
	}
}

func TestEmailSender_Send_implicitTLS(t *testing.T) {
	cert := testTLSCertificate(t)
	srv := startFakeSMTP(t, fakeSMTPOptions{implicitTLS: true, tlsCert: cert})
	defer srv.Close()

	e := &emailSender{
		host:       "127.0.0.1",
		from:       "groot@example.com",
		recipients: []string{"ops@example.com"},
		skipVerify: true,
	}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(srv.port))
	msg := buildEmailMessage(e.from, e.recipients, "implicit tls body")
	if err := e.sendImplicitTLS(addr, nil, msg); err != nil {
		t.Fatal(err)
	}
	if len(srv.messages()) != 1 {
		t.Fatalf("messages=%v", srv.messages())
	}
}

func TestEmailSender_Send_useTLS(t *testing.T) {
	cert := testTLSCertificate(t)
	srv := startFakeSMTP(t, fakeSMTPOptions{implicitTLS: true, tlsCert: cert})
	defer srv.Close()

	e := &emailSender{
		host:       "127.0.0.1",
		port:       srv.port,
		from:       "groot@example.com",
		recipients: []string{"ops@example.com"},
		useTLS:     true,
		skipVerify: true,
	}
	if err := e.Send(context.Background(), "tls via Send()"); err != nil {
		t.Fatal(err)
	}
}

func TestEmailSender_Send_port587STARTTLS(t *testing.T) {
	cert := testTLSCertificate(t)
	ln, err := net.Listen("tcp", "127.0.0.1:587")
	if err != nil {
		t.Skip("127.0.0.1:587 unavailable:", err)
	}
	srv := &fakeSMTPServer{
		t:    t,
		ln:   ln,
		port: 587,
		opts: fakeSMTPOptions{startTLS: true, tlsCert: cert},
	}
	go srv.acceptLoop()
	t.Cleanup(func() { srv.Close() })

	e := &emailSender{
		host:       "127.0.0.1",
		port:       587,
		from:       "groot@example.com",
		recipients: []string{"ops@example.com"},
		skipVerify: true,
	}
	if err := e.Send(context.Background(), "587 branch"); err != nil {
		t.Fatal(err)
	}
}

func TestEmailSender_Send_authRequired(t *testing.T) {
	srv := startFakeSMTP(t, fakeSMTPOptions{
		requireAuth: true,
		authUser:    "user",
		authPass:    "good",
	})
	defer srv.Close()

	ok := &emailSender{
		host:       "127.0.0.1",
		port:       srv.port,
		from:       "groot@example.com",
		recipients: []string{"ops@example.com"},
		username:   "user",
		password:   "good",
	}
	if err := ok.Send(context.Background(), "authed"); err != nil {
		t.Fatal(err)
	}

	bad := &emailSender{
		host:       "127.0.0.1",
		port:       srv.port,
		from:       "groot@example.com",
		recipients: []string{"ops@example.com"},
		username:   "user",
		password:   "bad",
	}
	if err := bad.Send(context.Background(), "nope"); err == nil || !strings.Contains(err.Error(), "auth") {
		t.Fatalf("err=%v", err)
	}
}

func TestAppendEmailSenders_multipleRecipients(t *testing.T) {
	out := appendEmailSenders(nil, emailCfgView{
		enabled: true,
		host:    "smtp.example.com",
		from:    "groot@example.com",
		to:      "one@x.com;two@y.com",
	})
	if len(out) != 1 {
		t.Fatalf("senders=%d", len(out))
	}
	es, ok := out[0].(*emailSender)
	if !ok || len(es.recipients) != 2 {
		t.Fatalf("recipients=%v", es.recipients)
	}
}

func TestFanOut_Notify_email(t *testing.T) {
	srv := startFakeSMTP(t, fakeSMTPOptions{})
	defer srv.Close()

	cfg := config.Config{}
	cfg.Notify.Email.Enabled = true
	cfg.Notify.Email.Host = "127.0.0.1"
	cfg.Notify.Email.Port = srv.port
	cfg.Notify.Email.From = "groot@example.com"
	cfg.Notify.Email.To = "oncall@example.com"

	f := NewFanOut(cfg)
	if len(f.senders) != 1 {
		t.Fatalf("senders=%d", len(f.senders))
	}
	sum := collector.Summary{Total: 1, Success: 1, Duration: time.Second, OutputDir: "/o", ArchivePath: "/a.tgz"}
	if err := f.Notify(context.Background(), sum); err != nil {
		t.Fatal(err)
	}
	if len(srv.messages()) != 1 || !strings.Contains(string(srv.messages()[0]), "GROOT") {
		t.Fatalf("messages=%v", srv.messages())
	}
}

type fakeSMTPOptions struct {
	startTLS    bool
	implicitTLS bool
	tlsCert     tls.Certificate
	requireAuth bool
	authUser    string
	authPass    string
}

type fakeSMTPServer struct {
	t      *testing.T
	ln     net.Listener
	port   int
	opts   fakeSMTPOptions
	mu     sync.Mutex
	msgs   [][]byte
	closed bool
}

func startFakeSMTP(t *testing.T, opts fakeSMTPOptions) *fakeSMTPServer {
	t.Helper()
	if opts.implicitTLS && opts.tlsCert.Certificate == nil {
		opts.tlsCert = testTLSCertificate(t)
	}
	if opts.startTLS && opts.tlsCert.Certificate == nil {
		opts.tlsCert = testTLSCertificate(t)
	}

	var ln net.Listener
	var err error
	addr := "127.0.0.1:0"
	if opts.implicitTLS {
		ln, err = tls.Listen("tcp", addr, &tls.Config{Certificates: []tls.Certificate{opts.tlsCert}})
	} else {
		ln, err = net.Listen("tcp", addr)
	}
	if err != nil {
		t.Fatal(err)
	}

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := net.LookupPort("tcp", portStr)
	if err != nil {
		t.Fatal(err)
	}

	s := &fakeSMTPServer{t: t, ln: ln, port: port, opts: opts}
	go s.acceptLoop()
	t.Cleanup(func() { s.Close() })
	return s
}

func (s *fakeSMTPServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	_ = s.ln.Close()
}

func (s *fakeSMTPServer) messages() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]byte, len(s.msgs))
	copy(out, s.msgs)
	return out
}

func (s *fakeSMTPServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return
			}
			s.t.Errorf("accept: %v", err)
			return
		}
		go s.handleConn(conn)
	}
}

func (s *fakeSMTPServer) handleConn(conn net.Conn) {
	defer conn.Close()
	if err := s.serveSMTP(conn); err != nil {
		s.t.Logf("smtp session: %v", err)
	}
}

func (s *fakeSMTPServer) serveSMTP(conn net.Conn) error {
	sess := &smtpSession{server: s, conn: conn, br: bufio.NewReader(conn)}
	sess.write = sess.writeLine
	if err := sess.writeLine("220 fake.local ESMTP"); err != nil {
		return err
	}
	for !sess.done {
		line, err := sess.readLine()
		if err != nil {
			return err
		}
		if sess.inData {
			if err := sess.onDataLine(line); err != nil {
				return err
			}
			continue
		}
		if err := sess.onCommand(line); err != nil {
			return err
		}
	}
	return nil
}

type smtpSession struct {
	server *fakeSMTPServer
	conn   net.Conn
	br     *bufio.Reader
	write  func(string) error

	authed  bool
	inData  bool
	done    bool
	dataBuf strings.Builder
}

func (sess *smtpSession) readLine() (string, error) {
	line, err := sess.br.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func (sess *smtpSession) writeLine(line string) error {
	_, err := sess.conn.Write([]byte(line + "\r\n"))
	return err
}

func (sess *smtpSession) onDataLine(line string) error {
	if line == "." {
		sess.server.mu.Lock()
		sess.server.msgs = append(sess.server.msgs, []byte(sess.dataBuf.String()))
		sess.server.mu.Unlock()
		sess.inData = false
		sess.dataBuf.Reset()
		return sess.writeLine("250 OK")
	}
	if strings.HasPrefix(line, "..") {
		line = line[1:]
	}
	sess.dataBuf.WriteString(line)
	sess.dataBuf.WriteString("\r\n")
	return nil
}

func (sess *smtpSession) onCommand(line string) error {
	upper := strings.ToUpper(line)
	switch {
	case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
		return sess.onHelo()
	case upper == "STARTTLS":
		return sess.onStartTLS()
	case strings.HasPrefix(upper, "AUTH PLAIN"):
		return sess.onAuthPlain(line)
	case strings.HasPrefix(upper, "MAIL FROM"):
		return sess.onMailFrom()
	case strings.HasPrefix(upper, "RCPT TO"):
		return sess.writeLine("250 OK")
	case upper == "DATA":
		if err := sess.writeLine("354 End data with <CR><LF>.<CR><LF>"); err != nil {
			return err
		}
		sess.inData = true
		return nil
	case upper == "QUIT":
		_ = sess.writeLine("221 Bye")
		sess.done = true
		return nil
	case upper == "RSET":
		sess.inData = false
		sess.dataBuf.Reset()
		_ = sess.writeLine("250 OK")
		return nil
	default:
		_ = sess.writeLine("502 not implemented")
		return nil
	}
}

func (sess *smtpSession) onHelo() error {
	if err := sess.writeLine("250-fake.local"); err != nil {
		return err
	}
	if sess.server.opts.startTLS {
		if err := sess.writeLine("250-STARTTLS"); err != nil {
			return err
		}
	}
	if sess.server.opts.requireAuth {
		if err := sess.writeLine("250-AUTH PLAIN"); err != nil {
			return err
		}
	}
	return sess.writeLine("250 OK")
}

func (sess *smtpSession) onStartTLS() error {
	if !sess.server.opts.startTLS {
		_ = sess.writeLine("502 not implemented")
		return nil
	}
	if err := sess.writeLine("220 Ready to start TLS"); err != nil {
		return err
	}
	tlsConn := tls.Server(sess.conn, &tls.Config{Certificates: []tls.Certificate{sess.server.opts.tlsCert}})
	if err := tlsConn.Handshake(); err != nil {
		return err
	}
	sess.conn = tlsConn
	sess.br = bufio.NewReader(tlsConn)
	sess.write = sess.writeLine
	return nil
}

func (sess *smtpSession) onAuthPlain(line string) error {
	if !sess.server.opts.requireAuth {
		_ = sess.writeLine("502 not implemented")
		return nil
	}
	if !checkPlainAuth(line, sess.server.opts.authUser, sess.server.opts.authPass) {
		_ = sess.writeLine("535 auth failed")
		sess.authed = false
		return nil
	}
	sess.authed = true
	return sess.writeLine("235 OK")
}

func (sess *smtpSession) onMailFrom() error {
	if sess.server.opts.requireAuth && !sess.authed {
		_ = sess.writeLine("530 auth required")
		return nil
	}
	return sess.writeLine("250 OK")
}

func checkPlainAuth(line, wantUser, wantPass string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	if !strings.HasPrefix(upper, "AUTH PLAIN") {
		return false
	}
	payload := strings.TrimSpace(line[len("AUTH PLAIN"):])
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return false
	}
	parts := strings.Split(string(raw), "\x00")
	if len(parts) < 3 {
		return false
	}
	return parts[1] == wantUser && parts[2] == wantPass
}

func testTLSCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}
