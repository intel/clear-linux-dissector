DISCONTINUATION OF PROJECT

This project will no longer be maintained by Intel.

Intel has ceased development and contributions including, but not limited to, maintenance, bug fixes, new releases, or updates, to this project.  

Intel no longer accepts patches to this project.

If you have an ongoing need to use this project, are interested in independently developing it, or would like to maintain patches for the open source software community, please create your own fork of this project.  

Contact: webadmin@linux.intel.com
# clr-dissector
Tools for extracting information out of a Clear Linux release

### Build and Install

````
make
make DESTDIR=some/path install
````

If no DESTDIR is specified then the binaries will be installed in ~/.gopath/bin

#### dissector

The dissector utility takes a list of bundles, resolves those to a full list of packages (including package deps), translates that to source rpms, downloads the source rpms and then extracts the content.

````
$ dissector --help
USAGE for dissector
  -bundles_url string
    	Base URL for downloading release archives of clr-bundles (default "https://github.com/clearlinux/clr-bundles")
  -clear_version int
    	Clear Linux version (default -1)
  -repo_url string
    	Base URL downloading releases (default "https://cdn.download.clearlinux.org")

$ dissector service-os
Downloading 24320/srpms/certifi-2018.4.16-47.src.rpm... 163 kB complete         
Downloading 24320/srpms/libtasn1-4.13-30.src.rpm... 1.9 MB complete             
Downloading 24320/srpms/libXdmcp-1.1.2-14.src.rpm... 342 kB complete 
<snip>
Extracting 24320/srpms/tk-8.6.8-18.src.rpm to 24320/source/tk-8.6.8-18...
Extracting 24320/srpms/certifi-2018.4.16-47.src.rpm to 24320/source/certifi-2018.4.16-47...
Extracting 24320/srpms/libtasn1-4.13-30.src.rpm to 24320/source/libtasn1-4.13-30...
<snip>
````

#### image2bundles

The image2bundles utility will look up an image definition file from the update stream and extract the bundles used to create the image.  If the command is run from a Clear Linux installation then it will by default use the installed version and update stream URL.  Both the version info and the base URL can be overriden with command line options.

````
$ image2bundles --help
USAGE for image2bundles
  -n string
    	Name of Clear Linux image
  -u string
    	Base URL for Clear repository (default "https://cdn.download.clearlinux.org/releases")
  -v int
    	Clear Linux version (default -1)
$ image2bundles -n service-os 
openssh-server
os-core-update
os-core
service-os
software-defined-cockpit

````
#### bundles2packages

The bundles2packages utility takes a list of bundles and returns the full list of packages (both directly listed in the bundle specification and the resulting package dependencies)

````
$ bundles2packages --help
USAGE for bundles2packages
  -clear_version int
    	Clear Linux version (default -1)
  -url string
    	Base URL for downloading release archives of clr-bundles (default "https://github.com/clearlinux/clr-bundles")

$ bundles2packages service-os glibc-lib-avx2
linux-firmware-ipu4
linux-pk414-sos
mcelog
iasimage
clr-boot-manager
shim
linux-firmware
python3-core
virtualenv
syslinux
pip
acrn-hypervisor
pycodestyle
python3-dev
setuptools

````
#### downloadrepo

The downloadrepo will download the repo metadata for a specific Clear Linux release.

````
$ downloadrepo --help
USAGE for downloadrepo
  -clear_version int
    	Clear Linux version (default -1)
  -url string
    	Base URL downloading releases (default "https://cdn.download.clearlinux.org")

$ downloadrepo
Downloading 24320/repodata/primary.sqlite.xz... 2.8 MB complete                 
Downloading 24320/repodata/filelist.sqlite.xz... 4.1 MB complete                
Downloading 24320/repodata/other.sqlite.xz... 461 kB complete                   
Downloading 24320/repodata/comps.xml.xz... 728 B complete
````

#### downloadpackages

````
$ downloadpackages --help
USAGE for downloadpackages
  -clear_version int
    	Clear Linux version (default -1)
  -skip
    	Skip downloading any source rpm files
  -url string
    	Base URL for downloading release source rpms (default "https://cdn.download.clearlinux.org")
$ downloadpackages weston
Downloading 24320/source/weston-4.0.0-17.src.rpm... 1.3 MB complete

````

#### packages2source

The packages2source utility takes a list of packages and returns a list of source rpm URLs.  The utility does not expand the list to include package dependencies.

````
$ packages2source --help
USAGE for packages2source
  -clear_version int
        Clear Linux version (default -1)
  -repo_url string
        Base URL downloading releases (default "https://cdn.download.clearlinux.org")

$ packages2source libc6
https://cdn.download.clearlinux.org/releases/24330/clear/source/SRPMS/glibc-2.27-187.src.rpm

````

#### Piping the utilites together

Each command can either take arguments on the commandline or piped in from stdin.  The following example shows how to initiate a download of all source packages for the currently installed Clear Linux version (where a specific version would need to be passed if this was run on some other OS) for the standard "service-os" image.

````
image2bundles -n service-os|bundles2packages|downloadpackages
Downloading 24320/source/libthai-0.1.28-6.src.rpm... 425 kB complete
Downloading 24320/source/shim-12-13.src.rpm... 1.0 MB complete
<snip>
````


