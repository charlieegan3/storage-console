package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/charlieegan3/storage-console/pkg/utils"
)

func InitMinio(ctx context.Context, t *testing.T) (minioClient *minio.Client, cleanup func() error, err error) {
	p, err := utils.FreePort(9093)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find free port: %s", err)
	}

	consolePort, err := utils.FreePort(9094)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find free port for console: %s", err)
	}

	t.Logf("Minio http://localhost:%d", p)
	t.Logf("Minio Console: http://localhost:%d", consolePort)

	adminUser := "minioadmin"
	adminPassword := "minioadmin"

	req := testcontainers.ContainerRequest{
		Image: "quay.io/minio/minio:RELEASE.2024-05-01T01-11-10Z",
		Cmd: []string{
			"server",
			"/data",
			"--address",
			fmt.Sprintf(":%d", p),
			"--console-address",
			fmt.Sprintf(":%d", consolePort),
		},
		ExposedPorts: []string{
			fmt.Sprintf("%d:%d", p, p),
			fmt.Sprintf("%d:%d", consolePort, consolePort),
		},
		Env: map[string]string{
			"MINIO_ROOT_USER":     adminUser,
			"MINIO_ROOT_PASSWORD": adminPassword,
		},
		WaitingFor: wait.ForLog("1 Online"),
	}
	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("could not start minio: %s", err)
	}
	cleanupFunc := func() error {
		if err := minioContainer.Terminate(ctx); err != nil {
			return fmt.Errorf("could not terminate minio: %s", err)
		}

		return nil
	}

	endpoint := fmt.Sprintf("localhost:%d", p)
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(adminUser, adminPassword, ""),
		Secure: false,
	})
	if err != nil {
		return nil, cleanupFunc, fmt.Errorf("could not create minio client: %s", err)
	}

	retries := 5
	for {
		_, err = minioClient.ListBuckets(ctx)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
		retries--
		if retries == 0 {
			return nil, cleanupFunc, fmt.Errorf("could not connect to minio before timeout: %s", err)
		}
	}

	return minioClient, cleanupFunc, nil
}
