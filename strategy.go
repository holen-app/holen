package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/kr/pretty"
	"github.com/pkg/errors"
)

var NoCheckSums error = fmt.Errorf("No Checksums")

type HashMismatch struct {
	algo     string
	checksum string
	hash     string
}

func (hm HashMismatch) Error() string {
	return fmt.Sprintf("using %s, expected %s and got %s", hm.algo, hm.checksum, hm.hash)
}

type StrategyCommon struct {
	System
	Logger
	ConfigGetter
	Downloader
	Runner
}

func (sc *StrategyCommon) Templater(version string, osArchData map[string]map[string]string, system System) Templater {
	archKey := fmt.Sprintf("%s_%s", system.OS(), system.Arch())
	sc.Debugf("Arch key: %s", archKey)
	value := osArchData[archKey]
	return Templater{
		Version:    version,
		OS:         system.OS(),
		Arch:       system.Arch(),
		OSArch:     archKey,
		OSArchData: value,
	}
}

type Strategy interface {
	Run([]string) error
}

type DockerData struct {
	Name        string
	Desc        string
	Version     string                       `yaml:"version"`
	Image       string                       `yaml:"image"`
	MountPwd    bool                         `yaml:"mount_pwd"`
	MountPwdAs  string                       `yaml:"mount_pwd_as"`
	DockerConn  bool                         `yaml:"docker_conn"`
	Interactive bool                         `yaml:"interactive"`
	Terminal    string                       `yaml:"terminal"`
	PidHost     bool                         `yaml:"pid_host"`
	RunAsUser   bool                         `yaml:"run_as_user"`
	OSArchData  map[string]map[string]string `yaml:"os_arch_map"`
}

type DockerStrategy struct {
	*StrategyCommon
	Data DockerData
}

type BinaryData struct {
	Name       string
	Desc       string
	Version    string                       `yaml:"version"`
	BaseURL    string                       `yaml:"base_url"`
	UnpackPath string                       `yaml:"unpack_path"`
	OSArchData map[string]map[string]string `yaml:"os_arch_map"`
}

type BinaryStrategy struct {
	*StrategyCommon
	Data BinaryData
}

func (ds DockerStrategy) Run(extraArgs []string) error {
	// skip if docker not found
	if !ds.CheckCommand("docker", []string{"version"}) {
		ds.Debugf("skipping, docker not available")
		return &SkipError{"docker not available"}
	}

	temp := ds.Templater(ds.Data.Version, ds.Data.OSArchData, ds.System)
	ds.Debugf("templater: %# v", pretty.Formatter(temp))

	image, err := temp.Template(ds.Data.Image)
	if err != nil {
		return errors.Wrap(err, "unable to template image name")
	}

	// TODO: add flag to force pulling image again
	// err = ds.PullDockerImage(image)
	// if err != nil {
	// 	return errors.Wrap(err, "can't pull image")
	// }

	args := []string{"run"}
	if ds.Data.Interactive {
		args = append(args, "-i")
	}
	// TODO: prompt the user for permission to do more invasive docker binding
	if ds.Data.DockerConn {
		args = append(args, "-v", "/var/run/docker.sock:/var/run/docker.sock")
	}
	if ds.Data.PidHost {
		args = append(args, "--pid", "host")
	}
	if len(ds.Data.MountPwdAs) > 0 {
		wd, _ := os.Getwd()
		args = append(args, "--volume", fmt.Sprintf("%s:%s", wd, ds.Data.MountPwdAs))
	}
	if ds.Data.MountPwd {
		wd, _ := os.Getwd()
		args = append(args, "--volume", fmt.Sprintf("%s:%s", wd, wd))
	}
	if ds.Data.RunAsUser {
		args = append(args, "-u", fmt.Sprintf("%d:%d", ds.UID(), ds.GID()))
	}
	if ds.Data.Terminal != "" {
		// TODO: support 'auto' mode that autodetects if tty is present
		if ds.Data.Terminal == "always" {
			args = append(args, "-t")
		}
	}
	args = append(args, "--rm", image)
	args = append(args, extraArgs...)

	err = ds.RunCommand("docker", args)
	if err != nil {
		return errors.Wrap(err, "can't run image")
	}

	return nil
}

func (bs BinaryStrategy) localHolenPath() (string, error) {
	var holenPath string
	if configDataPath, err := bs.Get("holen.datapath"); err == nil && len(configDataPath) > 0 {
		holenPath = configDataPath
	} else if xdgDataHome := os.Getenv("XDG_DATA_HOME"); len(xdgDataHome) > 0 {
		holenPath = filepath.Join(xdgDataHome, "holen")
	} else {
		var home string
		if home = os.Getenv("HOME"); len(home) == 0 {
			return "", fmt.Errorf("$HOME environment variable not found")
		}
		holenPath = filepath.Join(home, ".local", "share", "holen")
	}
	os.MkdirAll(holenPath, 0755)

	return holenPath, nil
}

func (bs BinaryStrategy) DownloadPath() (string, error) {
	var downloadPath string
	if configDownloadPath, err := bs.Get("binary.download"); err == nil && len(configDownloadPath) > 0 {
		downloadPath = configDownloadPath
	} else {
		holenPath, err := bs.localHolenPath()
		if err != nil {
			return "", errors.Wrap(err, "unable to get holen data path")
		}
		downloadPath = filepath.Join(holenPath, "bin")
	}
	os.MkdirAll(downloadPath, 0755)

	return downloadPath, nil
}

func (bs BinaryStrategy) TempPath() (string, error) {
	var tempPath string

	holenPath, err := bs.localHolenPath()
	if err != nil {
		return "", errors.Wrap(err, "unable to get holen data path")
	}

	tempPath = filepath.Join(holenPath, "tmp")
	os.MkdirAll(tempPath, 0755)

	return tempPath, nil
}

func (bs BinaryStrategy) Run(args []string) error {
	temp := bs.Templater(bs.Data.Version, bs.Data.OSArchData, bs.System)
	bs.Debugf("templater: %# v", pretty.Formatter(temp))

	dlURL, err := temp.Template(bs.Data.BaseURL)
	if err != nil {
		return errors.Wrap(err, "unable to template url")
	}

	downloadPath, err := bs.DownloadPath()
	if err != nil {
		return errors.Wrap(err, "unable to find download path")
	}
	binName := fmt.Sprintf("%s--%s", bs.Data.Name, bs.Data.Version)
	localPath := filepath.Join(downloadPath, binName)

	if !bs.FileExists(localPath) {
		var binPath, sumPath string

		tempPath, err := bs.TempPath()
		tempdir, err := ioutil.TempDir(tempPath, "holen")
		if err != nil {
			return errors.Wrap(err, "unable to make temporary directory")
		}
		defer os.RemoveAll(tempdir)
		if len(bs.Data.UnpackPath) > 0 {
			u, err := url.Parse(dlURL)
			if err != nil {
				return errors.Wrap(err, "unable to parse url")
			}

			fileName := path.Base(u.Path)
			archPath := filepath.Join(tempdir, fileName)
			unpackedPath := filepath.Join(tempdir, "unpacked")

			bs.UserMessage("Downloading %s...\n", dlURL)
			err = bs.DownloadFile(dlURL, archPath)
			if err != nil {
				return errors.Wrap(err, "can't download archive")
			}

			err = bs.UnpackArchive(archPath, unpackedPath)
			if err != nil {
				return errors.Wrap(err, "unable to unpack archive")
			}

			unpackPath, err := temp.Template(bs.Data.UnpackPath)
			if err != nil {
				return errors.Wrap(err, "unable to template unpack_path")
			}

			binPath = filepath.Join(unpackedPath, unpackPath)
			sumPath = archPath
		} else {
			binPath = filepath.Join(tempdir, binName)
			sumPath = binPath

			bs.UserMessage("Downloading %s...\n", dlURL)
			err = bs.DownloadFile(dlURL, binPath)
			if err != nil {
				return errors.Wrap(err, "can't download binary")
			}
		}

		err = bs.ChecksumBinary(sumPath)
		if err != nil {
			if err == NoCheckSums {
				bs.Debugf("skipping checksum, no checksums provided")
			} else {
				return errors.Wrap(err, "binary checksum failed")
			}
		}

		err = os.Rename(binPath, localPath)
		if err != nil {
			return errors.Wrap(err, "unable to move binary into position")
		}

		err = bs.MakeExecutable(localPath)
		if err != nil {
			return errors.Wrap(err, "unable to make binary executable")
		}

		os.RemoveAll(tempdir)
	}

	// TODO: add option to re-checksum the binary

	err = bs.RunCommand(localPath, args)
	if err != nil {
		return errors.Wrap(err, "can't run binary")
	}

	return nil
}

func (bs BinaryStrategy) ChecksumBinary(binaryPath string) error {
	data := bs.Data.OSArchData[fmt.Sprintf("%s_%s", bs.OS(), bs.Arch())]

	var algo, checksum string
	var ok bool
	if checksum, ok = data["sha256sum"]; ok {
		algo = "sha256"
	} else if checksum, ok = data["sha1sum"]; ok {
		algo = "sha1"
	} else if checksum, ok = data["md5sum"]; ok {
		algo = "md5"
	} else {
		return NoCheckSums
	}

	hash, err := hashFile(algo, binaryPath)
	if err != nil {
		return err
	} else if hash != checksum {
		return HashMismatch{algo, checksum, hash}
	}

	return nil
}
