#Molgif

Molgif is an easy-to-use tool for creating
[GIF](https://en.wikipedia.org/wiki/GIF) animations of molecules. Thanks to the
widespread support of [GIF](https://en.wikipedia.org/wiki/GIF) file format,
animations can be easily embedded into presentations, websites, wikipedia, and
so forth.

![caffeine](caffeine.gif)
![benzene](benzene.gif)

###Installation

Molgif works on Linux, BSD, OSX, Windows operating systems. Molgif requires
[Go](https://golang.org) version 1.6 or later. To compile, change into the
directory with molgif source code and type:

    go build

Molgif has no external dependencies and uses only [Go](https://golang.org)
standard library. Animation rendering is performed using ray-tracing. Rendering
is done in parallel using multiple CPUs available on the system.

###Usage examples

To see the list of available command line flags with documentation, type:

    molgif -help

Below is a list of sample animations along with the molgif command used to
create them.

######Caffeine molecule

`molgif caffeine.xyz`

![caffeine](caffeine.gif)

######Benzene molecule

`molgif -X -t 3 -a 0.2 benzene.xyz`

![benzene](benzene.gif)

######Carbon nanotube

`molgif -w 500 -t 5 nanotube.xyz`

![nanotube](nanotube.gif)

######Water molecule

`molgif -r 200 -g 200 water.xyz`

![water](water.gif)

######TNT molecule

`molgif -r 220 -g 220 tnt.xyz`

![tnt](tnt.gif)

######Fullerene molecule

`molgif -r 255 -g 255 -b 255 -t 5 fullerene.xyz`

![fullerene](fullerene.gif)

######Cluster of 8 water molecules

`molgif -l cluster.xyz`

![cluster](cluster.gif)
