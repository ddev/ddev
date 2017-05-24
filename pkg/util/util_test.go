package util_test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/system"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	// TestArchiveURL provides the URL of the test tar.gz asset
	TestArchiveURL = "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz"
	// TestArchivePath provides the path the test tar.gz asset is downloaded to
	TestArchivePath string
	// TestArchiveExtractDir is the directory in the archive to extract
	TestArchiveExtractDir = "wordpress-0.4.0/"
)

func TestMain(m *testing.M) {
	// prep assets for files tests
	testPath, err := ioutil.TempDir("", "filetest")
	util.CheckErr(err)
	testPath, err = filepath.EvalSymlinks(testPath)
	util.CheckErr(err)
	testPath = filepath.Clean(testPath)
	TestArchivePath = filepath.Join(testPath, "files.tar.gz")

	err = system.DownloadFile(TestArchivePath, TestArchiveURL)
	if err != nil {
		log.Fatalf("archive download failed: %s", err)
	}

	// prep docker container for docker util tests
	client := util.GetDockerClient()
	err = client.PullImage(docker.PullImageOptions{
		Repository: version.RouterImage,
		Tag:        version.RouterTag,
	}, docker.AuthConfiguration{})
	if err != nil {
		log.Fatal("failed to pull test image ", err)
	}

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Name: "envtest",
		Config: &docker.Config{
			Image: version.RouterImage + ":" + version.RouterTag,
			Labels: map[string]string{
				"com.docker.compose.service": "ddevrouter",
				"com.ddev.site-name":         "dockertest",
			},
			Env: []string{"HOTDOG=superior-to-corndog", "POTATO=future-fry"},
		},
	})
	if err != nil {
		log.Fatal("failed to start docker container ", err)
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
	err = os.Remove(TestArchivePath)
	if err != nil {
		log.Fatal("failed to remove test asset: ", err)
	}

	os.Exit(testRun)
}
