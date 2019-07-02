# AirBuild

AirBuild is a build system wrapper written in Go.

It creates an installation prefix, downloads packages, builds them using their build systems, and installs into the prefix.

## Getting AirBuild

```bash
$ go get github.com/aurashell/airwheel
```

## Using AirBuild

First, create a file called `airbuild.json` similar to the one below:

```json
{
  "wants": ["libairwheel"],
  "packages": {
    "ck": {
      "source": {
        "type": "git",
        "repository": "https://github.com/concurrencykit/ck",
        "revision": "{ck-revision}"
      },
      "tool": "autotools"
    },
    "icu": {
      "source": {
        "type": "git",
        "repository": "https://github.com/unicode-org/icu",
        "revision": "{icu-revision}"
      },
      "tool": "autotools",
      "where": "icu4c/source/"
    },
    "libairwheel": {
      "wants": ["ck", "icu"],
      "source": {
        "type": "git",
        "repository": "{libairwheel-source}",
        "revision": "{libairwheel-revision}"
      },
      "tool": "cmake"
    }
  }
}
```

Then, to provide values (their names are enclosed in `{}`), you can create simple JSON files like below:

```json
{
  "ck-revision": "0.7.0",
  "icu-revision": "release-64-2",
  "libairwheel-source": "{root}/libairwheel",
  "libairwheel-revision": "master"
}
```

After this, using AirBuild is as easy, as calling command like below in the same directory as `airbuild.json`:

```bash
$ airbuild [value-files.json...]
```

You will find build data in `airbuild-junk` directory and your install prefix at `airbuild-prefix`.

## Supported tools

* `cmake`
* `meson`
* `autotools`

## Supported sources

* `git`