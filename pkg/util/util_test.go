package util_test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/system"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	// The image here can be any image, it just has to exist so it can be used for labels, etc.
	TestRouterImage = "busybox"
	TestRouterTag   = "1"
)

func TestMain(m *testing.M) {
	// prep assets for files tests
	testPath, err := ioutil.TempDir("", "filetest")
	util.CheckErr(err)
	testPath, err = filepath.EvalSymlinks(testPath)
	util.CheckErr(err)
	testPath = filepath.Clean(testPath)
	TestTarArchivePath = filepath.Join(testPath, "files.tar.gz")

	err = system.DownloadFile(TestTarArchivePath, TestTarArchiveURL)
	if err != nil {
		log.Fatalf("archive download failed: %s", err)
	}

	// prep docker container for docker util tests
	client := util.GetDockerClient()

	err = client.PullImage(docker.PullImageOptions{
		Repository: TestRouterImage,
		Tag:        TestRouterTag,
	}, docker.AuthConfiguration{})
	if err != nil {
		log.Fatal("failed to pull test image ", err)
	}

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Name: "envtest",
		Config: &docker.Config{
			Image: TestRouterImage + ":" + TestRouterTag,
			Labels: map[string]string{
				"com.docker.compose.service": "ddevrouter",
				"com.ddev.site-name":         "dockertest",
			},
			Env: []string{"HOTDOG=superior-to-corndog", "POTATO=future-fry"},
		},
	})
	if err != nil {
		log.Fatal("failed to create/start docker container ", err)
	}

	testRun := m.Run()

	// teardown docker container from docker util tests
	err = client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container.ID,
		Force: true,
	})
	if err != nil {
		log.Fatal("failed to remove test container: ", err)
	}

	// cleanup test file
	err = os.Remove(TestTarArchivePath)
	if err != nil {
		log.Fatal("failed to remove test asset: ", err)
	}

	os.Exit(testRun)
}
