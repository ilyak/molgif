# Molgif

Molgif is an easy-to-use tool for creating
[GIF](https://en.wikipedia.org/wiki/GIF) animations of molecules. Thanks to the
widespread support of [GIF](https://en.wikipedia.org/wiki/GIF) file format,
animations can be easily embedded into presentations, websites, wikipedia, and
so forth.

![caffeine](caffeine.gif)
![fullerene](fullerene.gif)
![benzene](benzene.gif)

### Installation

Molgif works on Linux, BSD, OSX, Windows operating systems. Molgif requires
[Go](https://golang.org) version 1.5 or later. To download and compile
the code, issue:

    go get github.com/ilyak/molgif

Molgif has no external dependencies and uses only [Go](https://golang.org)
standard library. Animation rendering is performed using ray-tracing. Rendering
is done in parallel using multiple CPUs available on the system.

### Usage

List of available command-line flags with documentation can be obtained with
`molgif -help`:

    -X            rotate along x axis in reverse
    -Y            rotate along y axis in reverse
    -Z            rotate along z axis in reverse
    -a float      atom size (default 0.4)
    -b uint       background color blue component
    -d float      bond size (default 0.2)
    -e string     cpu profiling data file name
    -g uint       background color green component
    -h int        output image height (default 256)
    -l            hide molgif banner
    -o string     output file name
    -p            render image in png format
    -r uint       background color red component
    -t int        animation loop time in seconds (default 3)
    -w int        output image width (default 256)
    -x            rotate along x axis
    -y            rotate along y axis
    -z            rotate along z axis

### Samples

Below is a list of sample animations along with the molgif command used to
create them.

###### Caffeine molecule

    molgif caffeine.xyz

![caffeine](caffeine.gif)

###### Benzene molecule

    molgif -X -a 0.2 benzene.xyz

![benzene](benzene.gif)

###### Carbon nanotube

    molgif -w 500 -t 5 nanotube.xyz

![nanotube](nanotube.gif)

###### Water cluster

    molgif -Y -t 5 -r 80 -g 80 -b 80 water.xyz

![water](water.gif)

###### TNT molecule

    molgif -r 220 -g 220 tnt.xyz

![tnt](tnt.gif)

###### Fullerene molecule

    molgif -Y -t 8 -g 100 -b 100 -a 0.3 fullerene.xyz

![fullerene](fullerene.gif)

###### Adenine and Guanine nucleobases

    molgif -t 8 adenine.xyz
    molgif -t 4 guanine.xyz

![adenine](adenine.gif)
![guanine](guanine.gif)
