package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/convox/rack/client"
	"github.com/jhoonb/archivex"
	"github.com/spf13/viper"
)

func convoxClient() (*client.Client, error) {
	if !viper.IsSet("convox.host") || !viper.IsSet("convox.password") {
		return nil, fmt.Errorf("must set 'convox.host' and 'convox.password' in $HOME/.ranch.yaml")
	}

	host := viper.GetString("convox.host")
	password := viper.GetString("convox.password")
	version := "20151211151200"
	return client.New(host, password, version), nil
}

func ConvoxScale(appName, processName string, instances, memory int) error {
	client, err := convoxClient()

	if err != nil {
		return err
	}

	return client.SetFormation(appName, processName, strconv.Itoa(instances), strconv.Itoa(memory))
}

func ConvoxPromote(appName string, releaseId string) error {
	client, err := convoxClient()

	if err != nil {
		return err
	}

	_, err = client.PromoteRelease(appName, releaseId)

	if err != nil {
		return err
	}

	return nil
}

func ConvoxDeploy(appName string, buildDir string) (string, error) {
	client, err := convoxClient()

	if err != nil {
		return "", err
	}

	app, err := client.GetApp(appName)

	if err != nil {
		return "", err
	}

	switch app.Status {
	case "creating":
		return "", fmt.Errorf("app is still creating: %s", appName)
	case "running", "updating":
	default:
		return "", fmt.Errorf("unable to build app: %s", appName)
	}

	tar, err := createTarball(buildDir)

	if err != nil {
		return "", err
	}

	cache := true
	config := "docker-compose.yml"

	build, err := client.CreateBuildSource(appName, tar, cache, config)

	if err != nil {
		return "", err
	}

	return finishBuild(client, appName, build)
}

func createTarball(buildDir string) ([]byte, error) {
	tmpDir, err := ioutil.TempDir("", "ranch")
	defer os.RemoveAll(tmpDir)
	fmt.Println(tmpDir)

	if err != nil {
		return nil, err
	}

	tgzfile := path.Join(tmpDir, "build.tar.gz")

	tar := new(archivex.TarFile)

	err = tar.Create(tgzfile)
	if err != nil {
		return nil, err
	}

	err = tar.AddAll(buildDir, false)
	if err != nil {
		return nil, err
	}

	err = tar.Close()
	if err != nil {
		return nil, err
	}

	return ioutil.ReadFile(tgzfile)
}

func finishBuild(client *client.Client, appName string, build *client.Build) (string, error) {

	if build.Id == "" {
		return "", fmt.Errorf("unable to fetch build id")
	}

	reader, writer := io.Pipe()
	go io.Copy(os.Stdout, reader)
	err := client.StreamBuildLogs(appName, build.Id, writer)

	if err != nil {
		return "", err
	}

	return waitForBuild(client, appName, build.Id)
}

func waitForBuild(client *client.Client, appName, buildId string) (string, error) {
	for {
		build, err := client.GetBuild(appName, buildId)

		if err != nil {
			return "", err
		}

		switch build.Status {
		case "complete":
			return build.Release, nil
		case "error":
			return "", fmt.Errorf("%s build failed", appName)
		case "failed":
			return "", fmt.Errorf("%s build failed", appName)
		}

		time.Sleep(1 * time.Second)
	}

	return "", fmt.Errorf("can't get here")
}

func WaitForStatus(appName, status string) error {
	client, err := convoxClient()

	if err != nil {
		return err
	}

	for {
		app, err := client.GetApp(appName)

		if err != nil {
			return err
		}

		if app.Status == status {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("can't get here")
}

func ConvoxPs(appName string) (Processes, error) {
	client, err := convoxClient()

	if err != nil {
		return nil, err
	}

	convoxPs, err := client.GetProcesses(appName, false) // false == no process stats

	if err != nil {
		return nil, err
	}

	var ps Processes

	for _, v := range convoxPs {
		p := Process(v)
		ps = append(ps, p)
	}

	return ps, nil
}
