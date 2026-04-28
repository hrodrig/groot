package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"groot/pkg/config"
	"groot/pkg/kubemock"
)

func TestService_Run_minimal(t *testing.T) {
	cleanup := kubemock.Install(t)
	defer cleanup()

	out := t.TempDir()
	cfg := config.Config{
		Kubeconfig: "",
		OutputDir:  out,
		FilePrefix: "pfx",
		Collection: config.CollectionCfg{
			Timeout:             30 * time.Second,
			WorkerConcurrency:   2,
			Namespaces:          nil,
			Targets:             nil,
			ExtraKubectl:        nil,
			IncludePodLogs:      false,
			IncludePreviousLogs: false,
			IncludeNodeDetails:  false,
			PodLogTailLines:     0,
		},
	}

	svc := New(cfg)
	var started int
	svc.SetHooks(
		func(name string, args []string) { started++ },
		func(string) {},
		func(string, error) {},
	)

	sum, err := svc.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sum.ArchivePath == "" {
		t.Fatalf("archive: %q", sum.ArchivePath)
	}
	if !filepath.IsAbs(sum.ArchivePath) {
		t.Fatalf("expected absolute archive path: %q", sum.ArchivePath)
	}
	if _, err := os.Stat(sum.ArchivePath); err != nil {
		t.Fatal(err)
	}
	if sum.Total < 1 {
		t.Fatalf("expected jobs, got %#v", sum)
	}
	if started != sum.Total {
		t.Fatalf("hooks started=%d total=%d", started, sum.Total)
	}
}

func TestService_Run_fullFeatures(t *testing.T) {
	cleanup := kubemock.Install(t)
	defer cleanup()

	out := t.TempDir()
	cfg := config.Config{
		Kubeconfig: "",
		OutputDir:  out,
		FilePrefix: "pfx",
		Collection: config.CollectionCfg{
			Timeout:           45 * time.Second,
			WorkerConcurrency: 4,
			Namespaces:        []string{"default"},
			Targets: map[string]config.NamespaceTargets{
				"default": {Deployments: []string{"api"}},
			},
			ExtraKubectl:        []string{"get ns -o name"},
			IncludePodLogs:      true,
			IncludePreviousLogs: true,
			IncludeNodeDetails:  true,
			PodLogTailLines:     50,
		},
	}

	svc := New(cfg)
	svc.SetMessage("RCA  test")
	svc.SetHooks(
		func(string, []string) {},
		func(string) {},
		func(string, error) {},
	)

	sum, err := svc.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Failed > sum.Total {
		t.Fatalf("failed=%d total=%d", sum.Failed, sum.Total)
	}
	if sum.ArchivePath == "" {
		t.Fatal("no archive")
	}
}

func TestReadKubeMetadata_withMock(t *testing.T) {
	cleanup := kubemock.Install(t)
	defer cleanup()

	svc := New(config.Config{})
	meta, err := svc.ReadKubeMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if meta.Context != "ctx1" || meta.Cluster != "my-cluster" {
		t.Fatalf("%#v", meta)
	}
}

func TestService_Run_noKubectlInPath(t *testing.T) {
	out := t.TempDir()
	old := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", old) })
	_ = os.Setenv("PATH", t.TempDir())

	cfg := config.Config{
		OutputDir: out,
		Collection: config.CollectionCfg{
			Timeout:            5 * time.Second,
			WorkerConcurrency:  1,
			IncludePodLogs:     false,
			IncludeNodeDetails: false,
		},
	}
	svc := New(cfg)
	sum, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error (failed jobs are OK): %v", err)
	}
	if sum.Failed == 0 || sum.Total == 0 {
		t.Fatalf("expected kubectl failures when binary missing: total=%d failed=%d", sum.Total, sum.Failed)
	}
}

func TestService_buildJobs_nodeListFails(t *testing.T) {
	old := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", old) })
	_ = os.Setenv("PATH", t.TempDir())

	svc := New(config.Config{
		Collection: config.CollectionCfg{
			IncludeNodeDetails: true,
		},
	})
	_, err := svc.buildJobs(context.Background())
	if err == nil {
		t.Fatal("expected error listing nodes without kubectl")
	}
}

func TestService_resolvePodsForLogs_noTargetsUsesLines(t *testing.T) {
	cleanup := kubemock.Install(t)
	defer cleanup()

	svc := New(config.Config{
		Collection: config.CollectionCfg{
			IncludePodLogs: true,
			Targets:        nil,
		},
	})
	refs, err := svc.resolvePodsForLogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Name != "pod-a" {
		t.Fatalf("%#v", refs)
	}
}
