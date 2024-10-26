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
)

func InitMinio(ctx context.Context, t *testing.T) (minioClient *minio.Client, cleanup func() error, err error) {
	adminUser := "minioadmin"
	adminPassword := "minioadmin"

	req := testcontainers.ContainerRequest{
		Image: "quay.io/minio/minio:RELEASE.2024-05-01T01-11-10Z",
		Cmd: []string{
			"server",
			"/data",
			"--address",
			":9000",
			"--console-address",
			":9001",
		},
		ExposedPorts: []string{"9000", "9001"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     adminUser,
			"MINIO_ROOT_PASSWORD": adminPassword,
		},
		WaitingFor: wait.ForLog("1 Online").WithStartupTimeout(5 * time.Second),
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

	p, _ := minioContainer.MappedPort(ctx, "9000")

	t.Logf("Minio http://localhost:%s", p.Port())

	cp, _ := minioContainer.MappedPort(ctx, "9001")
	t.Logf("Minio Console: http://localhost:%s", cp.Port())

	endpoint := fmt.Sprintf("localhost:%s", p.Port())
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

		t.Log(err)

		time.Sleep(500 * time.Millisecond)

		retries--
		if retries == 0 {
			return nil, cleanupFunc, fmt.Errorf("could not connect to minio before timeout: %s", err)
		}
	}

	return minioClient, cleanupFunc, nil
}
