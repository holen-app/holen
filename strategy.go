package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

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

func (sc *StrategyCommon) CommonTemplateValues(version string, osArchData map[string]map[string]string, system System, values map[string]string) (map[string]string, error) {
	temp := sc.Templater(version, osArchData, system)
	sc.Debugf("templater: %# v", pretty.Formatter(temp))

	var err error
	for key, val := range values {
		values[key], err = temp.Template(val)
		if err != nil {
			return values, errors.Wrap(err, fmt.Sprintf("unable to template %s", key))
		}
	}
	return values, nil
}

type Strategy interface {
	Run([]string) error
	Inspect() error
	Version() string
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

func (ds DockerStrategy) TemplateValues(values map[string]string) (map[string]string, error) {
	return ds.CommonTemplateValues(ds.Data.Version, ds.Data.OSArchData, ds.System, values)
}

func (ds DockerStrategy) Run(extraArgs []string) error {
	// skip if docker not found
	if !ds.CheckCommand("docker", []string{"version"}) {
		ds.Debugf("skipping, docker not available")
		return &SkipError{"docker not available"}
	}

	templated, err := ds.TemplateValues(map[string]string{
		"Image": ds.Data.Image,
	})
	if err != nil {
		return err
	}

	image := templated["Image"]
	if err != nil {
		return errors.Wrap(err, "unable to template image name")
	}

	// TODO: add flag to force pulling image again
	// err = ds.PullDockerImage(image)
	// if err != nil {
	// 	return errors.Wrap(err, "can't pull image")
	// }

	args := ds.GenerateArgs(image, extraArgs)

	err = ds.RunCommand("docker", args)
	if err != nil {
		return errors.Wrap(err, "can't run image")
	}

	return nil
}

func (ds DockerStrategy) Version() string {
	return ds.Data.Version
}

func (ds DockerStrategy) GenerateArgs(image string, extraArgs []string) []string {
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

	return args
}

func (ds DockerStrategy) Inspect() error {
	templated, err := ds.TemplateValues(map[string]string{
		"Image": ds.Data.Image,
	})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error in templating docker version %s", ds.Data.Version))
	}

	ds.Stdoutf("Docker Strategy (version: %s):\n", ds.Data.Version)
	ds.Stdoutf("  final image: %s\n", templated["Image"])
	ds.Stdoutf("  final command: docker %s\n", strings.Join(ds.GenerateArgs(templated["Image"], []string{"[args]"}), " "))

	return nil
}

func (bs BinaryStrategy) localHolenPath() (string, error) {
	var holenPath string
	if configDataPath, err := bs.Get("holen.datapath"); err == nil && len(configDataPath) > 0 {
		holenPath = configDataPath
	} else if xdgDataHome := bs.Getenv("XDG_DATA_HOME"); len(xdgDataHome) > 0 {
		holenPath = filepath.Join(xdgDataHome, "holen")
	} else {
		var home string
		if home = bs.Getenv("HOME"); len(home) == 0 {
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

func (bs BinaryStrategy) TemplateValues(values map[string]string) (map[string]string, error) {
	return bs.CommonTemplateValues(bs.Data.Version, bs.Data.OSArchData, bs.System, values)
}

func (bs BinaryStrategy) Run(args []string) error {
	templated, err := bs.TemplateValues(map[string]string{
		"BaseURL":    bs.Data.BaseURL,
		"UnpackPath": bs.Data.UnpackPath,
	})
	if err != nil {
		return err
	}

	dlURL := templated["BaseURL"]

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
		if len(templated["UnpackPath"]) > 0 {
			u, err := url.Parse(dlURL)
			if err != nil {
				return errors.Wrap(err, "unable to parse url")
			}

			fileName := path.Base(u.Path)
			archPath := filepath.Join(tempdir, fileName)
			unpackedPath := filepath.Join(tempdir, "unpacked")

			bs.Stderrf("Downloading %s...\n", dlURL)
			err = bs.DownloadFile(dlURL, archPath)
			if err != nil {
				return errors.Wrap(err, "can't download archive")
			}

			err = bs.UnpackArchive(archPath, unpackedPath)
			if err != nil {
				return errors.Wrap(err, "unable to unpack archive")
			}

			unpackPath := templated["UnpackPath"]
			binPath = filepath.Join(unpackedPath, unpackPath)
			sumPath = archPath
		} else {
			binPath = filepath.Join(tempdir, binName)
			sumPath = binPath

			bs.Stderrf("Downloading %s...\n", dlURL)
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

func (bs BinaryStrategy) Inspect() error {
	templated, err := bs.TemplateValues(map[string]string{
		"BaseURL":    bs.Data.BaseURL,
		"UnpackPath": bs.Data.UnpackPath,
	})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error in templating binary version %s", bs.Data.Version))
	}

	bs.Stdoutf("Binary Strategy (version: %s):\n", bs.Data.Version)
	bs.Stdoutf("  final url: %s\n", templated["BaseURL"])
	if len(templated["UnpackPath"]) > 0 {
		bs.Stdoutf("  final unpack path: %s\n", templated["UnpackPath"])
	}
	algo, sum := bs.FindChecksumAlgoAndSum()
	if len(algo) > 0 {
		bs.Stdoutf("  checksum with %s: %s\n", algo, sum)
	}

	return nil
}

func (bs BinaryStrategy) FindChecksumAlgoAndSum() (string, string) {
	data := bs.Data.OSArchData[fmt.Sprintf("%s_%s", bs.OS(), bs.Arch())]

	var checksum string
	var ok bool
	if checksum, ok = data["sha256sum"]; ok {
		return "sha256", checksum
	} else if checksum, ok = data["sha1sum"]; ok {
		return "sha1", checksum
	} else if checksum, ok = data["md5sum"]; ok {
		return "md5", checksum
	}

	return "", ""
}

func (bs BinaryStrategy) ChecksumBinary(binaryPath string) error {
	algo, checksum := bs.FindChecksumAlgoAndSum()
	if len(algo) == 0 {
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

func (bs BinaryStrategy) Version() string {
	return bs.Data.Version
}

type CmdioData struct {
	Name       string
	Desc       string
	Version    string                       `yaml:"version"`
	Command    string                       `yaml:"command"`
	OSArchData map[string]map[string]string `yaml:"os_arch_map"`
}

type CmdioStrategy struct {
	*StrategyCommon
	Data CmdioData
}

func (cs CmdioStrategy) Version() string {
	return cs.Data.Version
}

func (cs CmdioStrategy) TemplateValues(values map[string]string) (map[string]string, error) {
	return cs.CommonTemplateValues(cs.Data.Version, cs.Data.OSArchData, cs.System, values)
}

func (cs CmdioStrategy) Inspect() error {
	templated, err := cs.TemplateValues(map[string]string{
		"Command": cs.Data.Command,
	})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error in templating cmdio version %s", cs.Data.Version))
	}

	cs.Stdoutf("Cmd.io Strategy (version: %s):\n", cs.Data.Version)
	cs.Stdoutf("  final command: %s\n", templated["Command"])

	return nil
}

func (cs CmdioStrategy) Run(args []string) error {
	templated, err := cs.TemplateValues(map[string]string{
		"Command": cs.Data.Command,
	})
	if err != nil {
		return err
	}

	err = cs.RunCommand("ssh", append([]string{"alpha.cmd.io", templated["Command"]}, args...))
	if err != nil {
		return errors.Wrap(err, "can't run cmdio session")
	}

	return nil
}
