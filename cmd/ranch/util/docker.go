package util

import (
	"net/url"
	"os"
	"strings"

	"github.com/goodeggs/platform/cmd/ranch/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
	"github.com/goodeggs/platform/cmd/ranch/Godeps/_workspace/src/github.com/spf13/viper"
)

func dockerClient() (*docker.Client, error) {
	return docker.NewClientFromEnv()
}

func DockerResolveImageName(imageName string) (string, error) {
	host := viper.GetString("docker.registry.host")

	registryUrl, err := url.Parse(host)

	if err != nil {
		return "", err
	}

	hostname := registryUrl.Host

	org := viper.GetString("docker.registry.org")

	if org == "" {
		return strings.Join([]string{hostname, imageName}, "/"), nil
	}

	return strings.Join([]string{hostname, org, imageName}, "/"), nil
}

func registryAuth() docker.AuthConfiguration {
	return docker.AuthConfiguration{
		Email:         viper.GetString("docker.registry.email"),
		Username:      viper.GetString("docker.registry.username"),
		Password:      viper.GetString("docker.registry.password"),
		ServerAddress: viper.GetString("docker.registry.host"),
	}
}

func DockerPush(imageNameWithTag string) error {
	parts := strings.Split(imageNameWithTag, ":")
	imageName, tag := parts[0], parts[1]

	client, err := dockerClient()

	if err != nil {
		return err
	}

	absoluteImageName, err := DockerResolveImageName(imageName)

	if err != nil {
		return err
	}

	opts := docker.PushImageOptions{
		Name:         absoluteImageName,
		Tag:          tag,
		OutputStream: os.Stdout,
	}

	err = client.PushImage(opts, registryAuth())

	if err != nil {
		return err
	}

	return nil
}

func DockerBuild(appDir string, imageName string) error {
	client, err := dockerClient()

	if err != nil {
		return err
	}

	absoluteImageName, err := DockerResolveImageName(imageName)

	if err != nil {
		return err
	}

	buildArgs := make([]docker.BuildArg, len(os.Environ()))
	for idx, item := range os.Environ() {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			buildArgs[idx].Name = parts[0]
			buildArgs[idx].Value = parts[1]
		}
	}

	opts := docker.BuildImageOptions{
		Name:         absoluteImageName,
		OutputStream: os.Stdout,
		ContextDir:   appDir,
		Pull:         true,
		BuildArgs:    buildArgs,
	}

	auth, err := docker.NewAuthConfigurationsFromDockerCfg()

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if auth != nil {
		opts.AuthConfigs = *auth
	}

	return client.BuildImage(opts)
}
