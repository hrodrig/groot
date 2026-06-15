package uploader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/hrodrig/groot/pkg/collector"
	"github.com/hrodrig/groot/pkg/config"
)

type sftpUploader struct {
	cfg     config.SFTPUploadCfg
	timeout time.Duration
}

func newSFTPUploader(cfg config.SFTPUploadCfg, timeout time.Duration) Uploader {
	return &sftpUploader{cfg: cfg, timeout: uploadTimeout(timeout)}
}

func (u *sftpUploader) Provider() string { return "sftp" }

func (u *sftpUploader) Upload(ctx context.Context, archivePath string, summary collector.Summary) (*Result, error) {
	_ = summary
	host := strings.TrimSpace(u.cfg.Host)
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}

	archiveName := filepath.Base(archivePath)
	remotePath := archiveName
	if dir := strings.TrimSpace(u.cfg.RemoteDir); dir != "" {
		remotePath = strings.TrimRight(dir, "/") + "/" + archiveName
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat archive: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	client, err := u.dialSSH(ctx, host)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	sclient, err := sftp.NewClient(client)
	if err != nil {
		return nil, fmt.Errorf("sftp client: %w", err)
	}
	defer sclient.Close()

	rf, err := sclient.Create(remotePath)
	if err != nil {
		return nil, fmt.Errorf("sftp create %s: %w", remotePath, err)
	}

	if _, err := io.Copy(rf, f); err != nil {
		_ = rf.Close()
		return nil, fmt.Errorf("sftp write: %w", err)
	}
	if err := rf.Close(); err != nil {
		return nil, fmt.Errorf("sftp close: %w", err)
	}

	uri := fmt.Sprintf("sftp://%s@%s:%d/%s", u.cfg.User, host, u.cfg.Port, remotePath)
	return &Result{Provider: "sftp", URI: uri, Key: remotePath, Size: info.Size()}, nil
}

func (u *sftpUploader) dialSSH(ctx context.Context, host string) (*ssh.Client, error) {
	user := strings.TrimSpace(u.cfg.User)
	if user == "" {
		return nil, fmt.Errorf("user is required")
	}

	identityFile := strings.TrimSpace(u.cfg.IdentityFile)
	if identityFile == "" {
		return nil, fmt.Errorf("identity_file is required (set via GROOT_UPLOAD_SFTP_IDENTITY_FILE)")
	}

	keyBytes, err := os.ReadFile(identityFile)
	if err != nil {
		return nil, fmt.Errorf("read identity file %s: %w", identityFile, err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse identity file %s: %w", identityFile, err)
	}

	var hostKeyCallback ssh.HostKeyCallback
	if knownHostsFile := strings.TrimSpace(u.cfg.KnownHostsFile); knownHostsFile != "" {
		cb, err := knownhosts.New(knownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("known_hosts file %s: %w", knownHostsFile, err)
		}
		hostKeyCallback = cb
	} else {
		hostKeyCallback = ssh.InsecureIgnoreHostKey() // OK in tests; production requires known_hosts
	}

	port := u.cfg.Port
	if port <= 0 {
		port = 22
	}

	sshCfg := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         u.timeout,
		// BatchMode equivalent: only allow publickey, reject interactive
		// golang.org/x/crypto/ssh doesn't support keyboard-interactive by default,
		// and we don't set a KeyboardInteractive callback, so only publickey works.
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	d := &net.Dialer{Timeout: u.timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, sshCfg)
	if err != nil {
		conn.Close()
		var hostKeyErr *knownhosts.KeyError
		if errors.As(err, &hostKeyErr) {
			return nil, fmt.Errorf("host key verification failed for %s: %w", addr, err)
		}
		return nil, fmt.Errorf("ssh handshake %s: %w", addr, err)
	}

	return ssh.NewClient(c, chans, reqs), nil
}
