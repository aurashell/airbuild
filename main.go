package main

import (
	"flag"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

}

func main() {
	flag.Parse()
	log.Info("Hello, World!")

	cwd, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}

	junkdir := path.Join(cwd, "airbuild-junk")
	prefixdir := path.Join(cwd, "airbuild-prefix")

	os.Setenv("PKG_CONFIG_PATH", path.Join(cwd, "airbuild-prefix", "lib", "pkgconfig")+":"+os.Getenv("PKG_CONFIG_PATH"))
	os.Setenv("LD_LIBRARY_PATH", path.Join(cwd, "airbuild-prefix", "lib")+":"+os.Getenv("LD_LIBRARY_PATH"))
	os.Setenv("LIBRARY_PATH", path.Join(cwd, "airbuild-prefix", "lib")+":"+os.Getenv("LIBRARY_PATH"))
	os.Setenv("PATH", path.Join(cwd, "airbuild-prefix", "bin")+":"+os.Getenv("PATH"))
	os.Setenv("AIRBUILD_JUNK", path.Join(cwd, "airbuild-junk"))
	os.Setenv("AIRBUILD_PREFIX", path.Join(cwd, "airbuild-prefix"))

	if _, err := os.Stat(junkdir); os.IsNotExist(err) {
		os.Mkdir(junkdir, os.FileMode(0755))
	}

	if _, err := os.Stat(prefixdir); os.IsNotExist(err) {
		os.Mkdir(prefixdir, os.FileMode(0755))
	}

	for _, valfilename := range flag.Args() {
		Repo.LoadValues(valfilename)
	}

	Repo.Load()
	Repo.GetAll()
	Repo.SetupAll()
}
