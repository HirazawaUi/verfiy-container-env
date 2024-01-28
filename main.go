package main

import (
	"context"
	"flag"
	"time"

	api "github.com/HirazawaUi/verfiy-container-env/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"k8s.io/klog/v2"
	kubeletutil "k8s.io/kubernetes/pkg/kubelet/util"
)

var (
	containerRuntimeEndpoint string
)

func init() {
	flag.StringVar(&containerRuntimeEndpoint, "endpoint", "unix:///run/containerd/containerd.sock", "")
}

func main() {
	flag.Parse()
	client := NewRuntimeClient(containerRuntimeEndpoint)

	ctx := context.TODO()
	runPodSandboxResp, err := client.RunPodSandbox(ctx, &api.RunPodSandboxRequest{
		Config: generatePodSandboxConfig(),
	})
	if err != nil {
		klog.Fatalf("Run PodSandbox failed, error: %v", err)
	}

	defer func() {
		_, err = client.RemovePodSandbox(ctx, &api.RemovePodSandboxRequest{
			PodSandboxId: runPodSandboxResp.PodSandboxId,
		})
		if err != nil {
			klog.Errorf("Remove PodSandbox failed, error: %v", err)
		}
	}()

	createContainerResp, err := client.CreateContainer(ctx, &api.CreateContainerRequest{
		PodSandboxId:  runPodSandboxResp.PodSandboxId,
		Config:        generateContainerConfig(),
		SandboxConfig: generatePodSandboxConfig(),
	})
	if err != nil {
		klog.Fatalf("Create container failed, error: %v", err)
	}
	defer func() {
		_, err = client.RemoveContainer(ctx, &api.RemoveContainerRequest{
			ContainerId: createContainerResp.ContainerId,
		})
		if err != nil {
			klog.Errorf("Remove container failed, error: %v", err)
		}
	}()

	_, err = client.StartContainer(ctx, &api.StartContainerRequest{
		ContainerId: createContainerResp.ContainerId,
	})
	if err != nil {
		klog.Fatalf("Start container failed, error: %v", err)
	}

	execResp, err := client.ExecSync(ctx, &api.ExecSyncRequest{
		ContainerId: createContainerResp.ContainerId,
		Cmd:         []string{"/bin/bash", "-c", "env | grep ASCII | wc -l"},
	})
	if err != nil {
		klog.Fatalf("Exec command in container failed, error: %v", err)
	}

	klog.Infof("The number of environment variables that have been set is %s", string(execResp.Stdout))
}

func generateContainerConfig() *api.ContainerConfig {
	var envs []*api.KeyValue
	for i := 33; i < 128; i++ {
		envKey := "ASCII" + string(i)
		envs = append(envs, &api.KeyValue{
			Key:   envKey,
			Value: string(i),
		})
	}

	return &api.ContainerConfig{
		Metadata: &api.ContainerMetadata{
			Name:    "env-demo-container",
			Attempt: 0,
		},
		Image: &api.ImageSpec{
			Image: "nginx:1.14.2",
		},
		Stdin:     false,
		StdinOnce: false,
		Tty:       false,
		Envs:      envs,
	}
}

func generatePodSandboxConfig() *api.PodSandboxConfig {
	return &api.PodSandboxConfig{
		Metadata: &api.PodSandboxMetadata{
			Name:      "env-demo",
			Namespace: "default",
			Attempt:   0,
		},
		Hostname:  "env-demo",
		DnsConfig: &api.DNSConfig{},
	}
}

func NewRuntimeClient(endpoint string) api.RuntimeServiceClient {
	addr, dialer, err := kubeletutil.GetAddressAndDialer(endpoint)
	if err != nil {
		klog.Errorf("Connect remote runtime failed, error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*16)))

	conn, err := grpc.DialContext(ctx, addr, dialOpts...)
	if err != nil {
		klog.Errorf("Connect remote runtime failed, error: %v", err)

	}

	runtimeClient := api.NewRuntimeServiceClient(conn)

	if _, err := runtimeClient.Version(ctx, &api.VersionRequest{}); err != nil {
		klog.Errorf("validate CRI v1 runtime API for endpoint %q: %v", endpoint, err)
	}

	return runtimeClient
}
