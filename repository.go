package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Repository - a package repository
type Repository struct {
	Wants    []string
	Packages map[string]Package
	Values   map[string]string
}

// Repo - a global repository instance
var Repo = Repository{
	Packages: make(map[string]Package),
	Values:   make(map[string]string),
}

// LoadValues - loads values from
func (r *Repository) LoadValues(filename string) {
	log.WithFields(log.Fields{"From": filename}).Info("Loading values")

	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		log.Panic(err)
	}

	var data map[string]string

	jsonParser := json.NewDecoder(file)
	jsonParser.Decode(&data)

	cwd, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}

	for k, v := range data {
		v = strings.ReplaceAll(v, "{root}", cwd)
		log.WithFields(log.Fields{"Key": k, "Value": v}).Info("Assign")
		r.Values[k] = v
	}
}

// ApplyValues - applies values defined in value files to manifest strings
func (r *Repository) ApplyValues(d string) string {
	for k, v := range r.Values {
		d = strings.ReplaceAll(d, "{"+k+"}", v)
	}
	return d
}

// Load - loads main manifest
func (r *Repository) Load() {
	log.Info("Loading main manifest")

	file, err := os.Open("./airbuild.json")
	defer file.Close()
	if err != nil {
		log.Panic(err)
	}

	var data map[string]interface{}

	jsonParser := json.NewDecoder(file)
	jsonParser.Decode(&data)

	r.Wants = []string{}

	for _, v := range data["wants"].([]interface{}) {
		r.Wants = append(r.Wants, v.(string))
	}

	log.WithFields(log.Fields{"Wants": r.Wants}).Info("We want")

	cores := runtime.NumCPU()

	for name, pkg := range data["packages"].(map[string]interface{}) {
		name := name
		pkg := pkg.(map[string]interface{})

		wants := []string{}

		if _, ok := pkg["wants"]; ok {
			for _, v := range pkg["wants"].([]interface{}) {
				wants = append(wants, v.(string))
			}
		}

		source := map[string]string{}

		for k, v := range pkg["source"].(map[string]interface{}) {
			source[k] = r.ApplyValues(v.(string))
		}

		cwd, err := os.Getwd()
		if err != nil {
			log.Panic(err)
		}

		tool := pkg["tool"].(string)

		sd := path.Join(cwd, "airbuild-junk", name+"-source")
		where := sd

		if w, ok := pkg["where"]; ok {
			where = path.Join(where, w.(string))
		}

		bd := where

		if tool == "cmake" || tool == "meson" {
			bd = path.Join(cwd, "airbuild-junk", name+"-build")
		}

		var getSteps []Step

		if source["type"] == "git" {
			rev := "master"
			if rev, ok := source["revision"]; ok {
				rev = rev
			}
			getSteps = []Step{
				Step{
					Wants: []string{},
					Commands: []string{
						"git clone " + source["repository"] + " -b " + rev + " " + sd,
					},
				},
			}
		} else if source["type"] == "link" {
			getSteps = []Step{
				Step{
					Wants: []string{},
					Commands: []string{
						"ln -s " + source["source"] + " " + sd,
					},
				},
			}
		}

		var buildSteps []Step
		var rebuildSteps []Step

		if tool == "autotools" {
			autogen := path.Join(where, "autogen.sh")
			configure := path.Join(where, "configure")
			makefile := path.Join(where, "Makefile")
			build0lock := path.Join(cwd, "airbuild-prefix", name+".build0lock")

			autogenstep := Step{
				Wants: []string{autogen},
				Commands: []string{
					autogen,
				},
			}

			configurestep := Step{
				Wants: []string{configure},
				Commands: []string{
					configure + " --prefix=" + path.Join(cwd, "airbuild-prefix"),
				},
			}

			makestep := Step{
				Wants: []string{makefile},
				Commands: []string{
					"make -j" + strconv.Itoa(cores*2),
					"touch " + build0lock,
				},
			}

			installstep := Step{
				Wants: []string{build0lock},
				Commands: []string{
					"make install",
					"touch " + path.Join(cwd, "airbuild-prefix", name+".buildlock"),
				},
			}

			buildSteps = []Step{
				autogenstep,
				configurestep,
				makestep,
				installstep,
			}

			rebuildSteps = []Step{
				configurestep,
				makestep,
				installstep,
			}
		} else if tool == "cmake" {
			cmakelists := path.Join(where, "CMakeLists.txt")
			makefile := path.Join(bd, "Makefile")
			build0lock := path.Join(cwd, "airbuild-prefix", name+".build0lock")

			cmakestep := Step{
				Wants: []string{cmakelists},
				Commands: []string{
					"cmake " + where + " -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=" + path.Join(cwd, "airbuild-prefix"),
				},
			}

			makestep := Step{
				Wants: []string{makefile},
				Commands: []string{
					"make -j" + strconv.Itoa(cores*2),
					"touch " + path.Join(cwd, "airbuild-prefix", name+".build0lock"),
				},
			}

			installstep := Step{
				Wants: []string{build0lock},
				Commands: []string{
					"make install",
					"touch " + path.Join(cwd, "airbuild-prefix", name+".buildlock"),
				},
			}

			buildSteps = []Step{
				cmakestep,
				makestep,
				installstep,
			}

			rebuildSteps = []Step{
				cmakestep,
				makestep,
				installstep,
			}
		} else if tool == "meson" {
			mesonbuild := path.Join(where, "meson.build")
			buildninja := path.Join(bd, "build.ninja")
			build0lock := path.Join(cwd, "airbuild-prefix", name+".build0lock")

			mesonstep := Step{
				Wants: []string{mesonbuild},
				Commands: []string{
					"meson " + where + " --buildtype=release --prefix " + path.Join(cwd, "airbuild-prefix"),
				},
			}

			remesonstep := Step{
				Wants: []string{mesonbuild},
				Commands: []string{
					"meson --reconfigure . " + where + " --buildtype=release --prefix " + path.Join(cwd, "airbuild-prefix"),
				},
			}

			ninjastep := Step{
				Wants: []string{buildninja},
				Commands: []string{
					"ninja",
					"touch " + build0lock,
				},
			}

			installstep := Step{
				Wants: []string{build0lock},
				Commands: []string{
					"ninja install",
					"touch " + path.Join(cwd, "airbuild-prefix", name+".buildlock"),
				},
			}

			buildSteps = []Step{
				mesonstep,
				ninjastep,
				installstep,
			}

			rebuildSteps = []Step{
				remesonstep,
				ninjastep,
				installstep,
			}
		}

		rpkg := Package{
			Name:         name,
			Wants:        wants,
			Source:       source,
			Tool:         tool,
			Where:        where,
			SourceDir:    sd,
			BuildDir:     bd,
			GetSteps:     getSteps,
			BuildSteps:   buildSteps,
			RebuildSteps: rebuildSteps,
			NoTouch:      false,
		}

		log.WithFields(log.Fields{"Package": name}).Info("New package")

		r.Packages[name] = rpkg
	}
}

func runCommand(s string, dir string) {
	log.WithFields(log.Fields{"Command": s}).Info("Executing a command")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
	}
	cmd := exec.Command("bash", "-c", "cd "+dir+" && "+s)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		log.Panic(err)
	}
}

// Get - gets a package
func (r *Repository) Get(name string) {
	pkg := r.Packages[name]
	cwd, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}
	if _, err := os.Stat(pkg.SourceDir); os.IsNotExist(err) {
		log.WithFields(log.Fields{"Package": name}).Info("Getting a package")
		useStep := func(i int, useStep interface{}) {
			check := func() bool {
				for _, w := range pkg.GetSteps[i].Wants {
					if _, err := os.Stat(w); os.IsNotExist(err) {
						return false
					}
				}
				return true
			}
			if !check() {
				useStep.(func(int, interface{}))(i-1, useStep)
				if !check() {
					log.WithFields(log.Fields{"Package": name}).Panic("Cannot get a package")
				}
			}
			for _, cmd := range pkg.GetSteps[i].Commands {
				runCommand(cmd, path.Join(cwd, "airbuild-junk"))
			}
		}
		useStep(len(pkg.GetSteps)-1, useStep)
	} else if findInStringSlice(r.Wants, name) && pkg.Source["type"] == "git" {
		runCommand("git pull", pkg.SourceDir)
		rev := "master"
		if rev, ok := pkg.Source["revision"]; ok {
			rev = rev
		}
		runCommand("git checkout "+rev, pkg.SourceDir)
	}
}

// GetAll - gets all packages
func (r *Repository) GetAll() {
	for k := range r.Packages {
		r.Get(k)
	}
}

func findInStringSlice(s []string, v string) bool {
	for _, a := range s {
		if a == v {
			return true
		}
	}
	return false
}

// Setup - sets up a package
func (r *Repository) Setup(name string) {
	pkg := r.Packages[name]
	if pkg.NoTouch {
		return
	}
	pkg.NoTouch = true
	for _, w := range pkg.Wants {
		r.Setup(w)
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}
	if _, err := os.Stat(path.Join(cwd, "airbuild-prefix", name+".buildlock")); os.IsNotExist(err) {
		log.WithFields(log.Fields{"Package": name}).Info("Setting up a package")
		useStep := func(i int, useStep interface{}) {
			log.Info(pkg.BuildSteps[i])
			check := func() bool {
				for _, w := range pkg.BuildSteps[i].Wants {
					if _, err := os.Stat(w); os.IsNotExist(err) {
						return false
					}
				}
				return true
			}
			if !check() {
				useStep.(func(int, interface{}))(i-1, useStep)
				if !check() {
					log.WithFields(log.Fields{"Package": name}).Panic("Cannot set up a package")
				}
			}
			for _, cmd := range pkg.BuildSteps[i].Commands {
				runCommand(cmd, pkg.BuildDir)
			}
		}
		useStep(len(pkg.BuildSteps)-1, useStep)
	} else if findInStringSlice(r.Wants, name) {
		log.WithFields(log.Fields{"Package": name}).Info("(Re)Setting up a package")
		useStep := func(i int, useStep interface{}) {
			check := func() bool {
				for _, w := range pkg.RebuildSteps[i].Wants {
					if _, err := os.Stat(w); os.IsNotExist(err) {
						return false
					}
				}
				return true
			}
			if !check() {
				log.WithFields(log.Fields{"Package": name}).Panic("Cannot (re)set up a package")
			}
			for _, cmd := range pkg.RebuildSteps[i].Commands {
				runCommand(cmd, pkg.BuildDir)
			}
		}
		for s := range pkg.RebuildSteps {
			useStep(s, useStep)
		}
	}
}

// SetupAll - sets up all packages
func (r *Repository) SetupAll() {
	for k := range r.Packages {
		r.Setup(k)
	}
}
