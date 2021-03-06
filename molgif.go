// (c) 2016 Ilya Kaliman. ISC license.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"math"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
)

type Mat struct {
	xx, xy, xz float32
	yx, yy, yz float32
	zx, zy, zz float32
}

func MatIdent() Mat {
	return Mat{
		1, 0, 0,
		0, 1, 0,
		0, 0, 1,
	}
}

func MatVec(m Mat, v Vec) Vec {
	return Vec{
		m.xx*v.x + m.xy*v.y + m.xz*v.z,
		m.yx*v.x + m.yy*v.y + m.yz*v.z,
		m.zx*v.x + m.zy*v.y + m.zz*v.z,
	}
}

func MatMat(m1 Mat, m2 Mat) Mat {
	return Mat{
		m1.xx*m2.xx + m1.xy*m2.yx + m1.xz*m2.zx,
		m1.xx*m2.xy + m1.xy*m2.yy + m1.xz*m2.zy,
		m1.xx*m2.xz + m1.xy*m2.yz + m1.xz*m2.zz,
		m1.yx*m2.xx + m1.yy*m2.yx + m1.yz*m2.zx,
		m1.yx*m2.xy + m1.yy*m2.yy + m1.yz*m2.zy,
		m1.yx*m2.xz + m1.yy*m2.yz + m1.yz*m2.zz,
		m1.zx*m2.xx + m1.zy*m2.yx + m1.zz*m2.zx,
		m1.zx*m2.xy + m1.zy*m2.yy + m1.zz*m2.zy,
		m1.zx*m2.xz + m1.zy*m2.yz + m1.zz*m2.zz,
	}
}

func (m *Mat) Scale(s float32) {
	m.xx *= s
	m.xy *= s
	m.xz *= s
	m.yx *= s
	m.yy *= s
	m.yz *= s
	m.zx *= s
	m.zy *= s
	m.zz *= s
}

func (m *Mat) Add(b Mat) {
	m.xx += b.xx
	m.xy += b.xy
	m.xz += b.xz
	m.yx += b.yx
	m.yy += b.yy
	m.yz += b.yz
	m.zx += b.zx
	m.zy += b.zy
	m.zz += b.zz
}

func MatRotX(angle float32) Mat {
	s64, c64 := math.Sincos(float64(angle))
	s := float32(s64)
	c := float32(c64)
	return Mat{
		1, 0, 0,
		0, c, s,
		0, -s, c,
	}
}

func MatRotY(angle float32) Mat {
	s64, c64 := math.Sincos(float64(angle))
	s := float32(s64)
	c := float32(c64)
	return Mat{
		c, 0, -s,
		0, 1, 0,
		s, 0, c,
	}
}

func MatRotZ(angle float32) Mat {
	s64, c64 := math.Sincos(float64(angle))
	s := float32(s64)
	c := float32(c64)
	return Mat{
		c, s, 0,
		-s, c, 0,
		0, 0, 1,
	}
}

func MatSkew(v Vec) Mat {
	return Mat{
		0, -v.z, v.y,
		v.z, 0, -v.x,
		-v.y, v.x, 0,
	}
}

// Returns rotation matrix that aligns a with b
// Vectors a and b must be normalized
func MatAlignRot(a, b Vec) Mat {
	v := VecCross(a, b)
	s := v.Len()
	c := VecDot(a, b)
	x := MatSkew(v)
	r := MatIdent()
	r.Add(x)
	x = MatMat(x, x)
	x.Scale((1.0 - c) / (s * s))
	r.Add(x)
	return r
}

type Vec struct {
	x, y, z float32
}

func (v *Vec) LenSq() float32 {
	return v.x*v.x + v.y*v.y + v.z*v.z
}

func (v *Vec) Len() float32 {
	return float32(math.Sqrt(float64(v.LenSq())))
}

func VecAdd(a, b Vec) Vec {
	a.Add(b)
	return a
}

func VecSub(a, b Vec) Vec {
	a.Sub(b)
	return a
}

func VecDot(a, b Vec) float32 {
	return a.x*b.x + a.y*b.y + a.z*b.z
}

func VecCross(a, b Vec) Vec {
	return Vec{
		a.y*b.z - a.z*b.y,
		a.z*b.x - a.x*b.z,
		a.x*b.y - a.y*b.x,
	}
}

func VecScale(v Vec, s float32) Vec {
	v.Scale(s)
	return v
}

func (v *Vec) Add(a Vec) {
	v.x += a.x
	v.y += a.y
	v.z += a.z
}

func (v *Vec) Sub(a Vec) {
	v.x -= a.x
	v.y -= a.y
	v.z -= a.z
}

func (v *Vec) Scale(s float32) {
	v.x *= s
	v.y *= s
	v.z *= s
}

func (v *Vec) Normalize() {
	v.Scale(1.0 / v.Len())
}

type Ray struct {
	orig, dir Vec // dir is normalized
}

type Material struct {
	diffuse color.RGBA
}

func TestLineSphere(ray *Ray, sp *Vec, sr float32) bool {
	lx := sp.x - ray.orig.x
	ly := sp.y - ray.orig.y
	lz := sp.z - ray.orig.z
	ll := lx*lx + ly*ly + lz*lz
	dl := ray.dir.x*lx + ray.dir.y*ly + ray.dir.z*lz
	return ll-dl*dl < sr*sr
}

type Shape interface {
	FastIntersect(Ray) bool
	Intersect(Ray) (float32, Vec, Vec)
	Material() Material
}

type Sphere struct {
	pos      Vec
	radius   float32
	material Material
}

func NewSphere(pos Vec, radius float32) Sphere {
	return Sphere{
		pos:    pos,
		radius: radius,
	}
}

func (s *Sphere) FastIntersect(ray Ray) bool {
	return TestLineSphere(&ray, &s.pos, s.radius)
}

func (s *Sphere) Intersect(ray Ray) (float32, Vec, Vec) {
	l := VecSub(s.pos, ray.orig)
	tca := VecDot(ray.dir, l)
	d2 := l.LenSq() - tca*tca
	r2 := s.radius * s.radius
	if d2 > r2 {
		return math.MaxFloat32, Vec{}, Vec{}
	}
	thc := float32(math.Sqrt(float64(r2 - d2)))
	t := tca - thc
	if t < 0 {
		t = tca + thc
	}
	p := ray.dir
	p.Scale(t)
	p.Add(ray.orig)
	n := p
	n.Sub(s.pos)
	n.Normalize()
	return t, p, n
}

func (s *Sphere) Material() Material {
	return s.material
}

type Cylinder struct {
	pos          Vec
	sphereRadius float32
	axis         Vec
	radius       float32
	halfz        float32
	material     Material
}

func NewCylinder(a, b Vec, r float32) Cylinder {
	c := Cylinder{}
	c.radius = r
	c.pos = VecAdd(a, b)
	c.pos.Scale(0.5)
	c.axis = VecSub(b, a)
	c.halfz = c.axis.Len() / 2
	c.sphereRadius = float32(math.Sqrt(float64(r*r + c.halfz*c.halfz)))
	c.axis.Normalize()
	return c
}

func (c *Cylinder) FastIntersect(ray Ray) bool {
	return TestLineSphere(&ray, &c.pos, c.sphereRadius)
}

func (c *Cylinder) Intersect(ray Ray) (float32, Vec, Vec) {
	rot := MatAlignRot(c.axis, Vec{0, 0, 1})
	ray.dir = MatVec(rot, ray.dir)
	ray.orig.Sub(c.pos)
	ray.orig = MatVec(rot, ray.orig)
	dd := ray.dir.x*ray.dir.x + ray.dir.y*ray.dir.y
	oo := ray.orig.x*ray.orig.x + ray.orig.y*ray.orig.y
	b := 2 * (ray.orig.x*ray.dir.x + ray.orig.y*ray.dir.y)
	d := b*b - 4*dd*(oo-c.radius*c.radius)
	if d < 0 {
		return math.MaxFloat32, Vec{}, Vec{}
	}
	d = float32(math.Sqrt(float64(d)))
	t := (-b - d) / (2 * dd)
	if t < 0 {
		t = (-b + d) / (2 * dd)
	}
	p := ray.dir
	p.Scale(t)
	p.Add(ray.orig)
	if p.z < -c.halfz || p.z > c.halfz {
		return math.MaxFloat32, Vec{}, Vec{}
	}
	n := Vec{p.x, p.y, 0}
	rot = MatAlignRot(Vec{0, 0, 1}, c.axis)
	p = MatVec(rot, p)
	p.Add(c.pos)
	n = MatVec(rot, n)
	n.Normalize()
	return t, p, n
}

func (c *Cylinder) Material() Material {
	return c.material
}

type View struct {
	width, height        int
	pos, look, right, up Vec
}

func NewView(width, height int, dist float32) *View {
	pos := Vec{0, 0, -dist}
	v := View{
		width:  width,
		height: height,
		pos:    pos,
		look:   Vec{0, 0, 1},
		right:  Vec{1, 0, 0},
		up:     Vec{0, 1, 0},
	}
	return &v
}

func (v *View) NewRay(x, y int) Ray {
	dx := float32(x-v.width/2) / float32(v.height)
	dy := float32(y-v.height/2) / float32(v.height)
	dir := v.look
	dir.Add(VecScale(v.right, dx))
	dir.Add(VecScale(v.up, dy))
	dir.Normalize()
	return Ray{v.pos, dir}
}

type Atom struct {
	name string
	pos  Vec
}

type Bond struct {
	a, b *Atom
}

type Molecule struct {
	atoms []*Atom
	bonds []*Bond
}

func (mol *Molecule) MoveToOrigin() {
	var c Vec
	for _, a := range mol.atoms {
		c.Add(a.pos)
	}
	c.Scale(1.0 / float32(len(mol.atoms)))
	for _, a := range mol.atoms {
		a.pos.Sub(c)
	}
}

func (mol *Molecule) MakeBonds() {
	const thresh = 1.6
	mol.bonds = nil
	for _, a := range mol.atoms {
		for _, b := range mol.atoms {
			if a.name == "H" && b.name == "H" {
				continue
			}
			pa := a.pos
			pb := b.pos
			if d := VecSub(pa, pb); d.LenSq() < thresh*thresh {
				bnd := Bond{a, b}
				mol.bonds = append(mol.bonds, &bnd)
			}
		}
	}
}

func (mol *Molecule) Rotate(rot Mat) {
	for _, a := range mol.atoms {
		a.pos = MatVec(rot, a.pos)
	}
}

func (sc *Scene) UpdateGeometry() {
	sc.shapes = nil
	for _, a := range sc.mol.atoms {
		p := NewSphere(a.pos, sc.atomSize)
		p.material = Material{Elements[a.name]}
		sc.shapes = append(sc.shapes, &p)
	}
	if sc.bondSize > 0.001 {
		for _, bnd := range sc.mol.bonds {
			mid := VecAdd(bnd.a.pos, bnd.b.pos)
			mid.Scale(0.5)
			a := NewCylinder(bnd.a.pos, mid, sc.bondSize)
			a.material = Material{Elements[bnd.a.name]}
			sc.shapes = append(sc.shapes, &a)
			b := NewCylinder(bnd.b.pos, mid, sc.bondSize)
			b.material = Material{Elements[bnd.b.name]}
			sc.shapes = append(sc.shapes, &b)
		}
	}
}

func NewMolecule(path string) (*Molecule, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	mol := new(Molecule)
	sc := bufio.NewScanner(f)
	sc.Scan() // skip atom count
	sc.Scan() // skip comment
	for sc.Scan() {
		var v Vec
		var s string
		fmt.Sscanf(sc.Text(), "%s %f %f %f", &s, &v.x, &v.y, &v.z)
		s = strings.Title(s)
		_, ok := Elements[s]
		if !ok {
			return nil, fmt.Errorf("unknown element: %s", s)
		}
		atom := Atom{s, v}
		mol.atoms = append(mol.atoms, &atom)
	}
	if err = sc.Err(); err != nil {
		return nil, err
	}
	if len(mol.atoms) == 0 {
		return nil, fmt.Errorf("%s: no atoms found", path)
	}
	mol.MakeBonds()
	mol.MoveToOrigin()
	return mol, nil
}

func DrawBanner(img *image.RGBA) {
	cl := color.RGBA{200, 200, 200, 255}
	pts := []string{
		"oo.oo.oooo.o....ooo.ooo.ooo",
		"o.o.o.o..o.o...o.....o..o..",
		"o...o.o..o.o...o.oo..o..oo.",
		"o...o.oooo.ooo..ooo.ooo.o..",
	}
	off := img.Bounds().Max
	off.X -= len(pts[0]) + 2
	off.Y -= len(pts) + 2
	for y, s := range pts {
		for x, c := range s {
			if c == 'o' {
				img.Set(off.X+x, off.Y+y, cl)
			}
		}
	}
}

type PointLight struct {
	pos Vec
}

type Scene struct {
	mol      *Molecule
	view     *View
	shapes   []Shape
	light    PointLight
	bg       color.RGBA
	banner   bool
	atomSize float32
	bondSize float32
}

func NewScene(mol *Molecule, w, h int) *Scene {
	var r float32
	for _, a := range mol.atoms {
		l := a.pos.Len()
		if l > r {
			r = l
		}
	}
	sc := Scene{
		mol:  mol,
		view: NewView(w, h, r+8),
	}
	sc.light = PointLight{Vec{1000, 500, -1000}}
	return &sc
}

func (sc *Scene) ComputePixel(x, y int) color.RGBA {
	pix := sc.bg
	var zmin float32 = math.MaxFloat32
	ray := sc.view.NewRay(x, y)
	for _, s := range sc.shapes {
		if !s.FastIntersect(ray) {
			continue
		}
		z, v, n := s.Intersect(ray)
		if z < zmin {
			zmin = z
			lp := sc.light.pos
			lp.Sub(v)
			lp.Normalize()
			dot := VecDot(lp, n)
			if dot < 0.0 {
				dot = 0.0
			}
			dot += 0.2 // add ambient light
			if dot > 1.0 {
				dot = 1.0
			}
			pix = s.Material().diffuse
			pix.R = uint8(float32(pix.R) * dot)
			pix.G = uint8(float32(pix.G) * dot)
			pix.B = uint8(float32(pix.B) * dot)
		}
	}
	return pix
}

func (sc *Scene) RenderTile(b image.Rectangle) image.Image {
	img := image.NewRGBA(b)
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			img.Set(x, y, sc.ComputePixel(x, y))
		}
	}
	return img
}

func (sc *Scene) Render() image.Image {
	const tileSize = 64
	bounds := image.Rect(0, 0, sc.view.width, sc.view.height)
	img := image.NewRGBA(bounds)
	np := runtime.NumCPU()
	ntilx := (bounds.Dx() + tileSize - 1) / tileSize
	ntily := (bounds.Dy() + tileSize - 1) / tileSize
	var wg sync.WaitGroup
	in := make(chan image.Rectangle)
	out := make(chan image.Image, ntilx*ntily)
	// render frame in parallel
	for i := 0; i < np; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range in {
				out <- sc.RenderTile(t)
			}
		}()
	}
	for i := 0; i < ntilx; i++ {
		for j := 0; j < ntily; j++ {
			x := i * tileSize
			y := j * tileSize
			in <- image.Rect(x, y, x+tileSize, y+tileSize)
		}
	}
	close(in)
	wg.Wait()
	close(out)
	for m := range out {
		draw.Draw(img, m.Bounds(), m, m.Bounds().Min, draw.Src)
	}
	if sc.banner {
		DrawBanner(img)
	}
	return img
}

func MakePaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	pm := image.NewPaletted(bounds, palette.Plan9)
	draw.FloydSteinberg.Draw(pm, bounds, img, image.ZP)
	return pm
}

func RenderAll(sc *Scene, loopTime int, rotvec [3]float32) *gif.GIF {
	const FPS = 50
	nframes := loopTime * FPS
	var sum float64
	for i := range rotvec {
		sum += math.Abs(float64(rotvec[i]))
	}
	ang := 2.0 * math.Pi / float32(nframes) / float32(math.Sqrt(sum))
	rot := MatIdent()
	rot = MatMat(rot, MatRotX(ang*rotvec[0]))
	rot = MatMat(rot, MatRotY(ang*rotvec[1]))
	rot = MatMat(rot, MatRotZ(ang*rotvec[2]))
	var g gif.GIF
	for i := 0; i < nframes; i++ {
		img := sc.Render()
		g.Image = append(g.Image, MakePaletted(img))
		g.Delay = append(g.Delay, 100/FPS)
		sc.mol.Rotate(rot)
		sc.UpdateGeometry()
		if i%10 == 0 {
			fmt.Print(".")
		}
	}
	fmt.Println()
	return &g
}

func main() {
	log.SetFlags(0)
	oFlag := flag.String("o", "", "output file name")
	eFlag := flag.String("e", "", "cpu profiling data file name")
	wFlag := flag.Int("w", 256, "output image width")
	hFlag := flag.Int("h", 256, "output image height")
	tFlag := flag.Int("t", 3, "animation loop time in seconds")
	xFlag := flag.Bool("x", false, "rotate along x axis")
	yFlag := flag.Bool("y", false, "rotate along y axis")
	zFlag := flag.Bool("z", false, "rotate along z axis")
	XFlag := flag.Bool("X", false, "rotate along x axis in reverse")
	YFlag := flag.Bool("Y", false, "rotate along y axis in reverse")
	ZFlag := flag.Bool("Z", false, "rotate along z axis in reverse")
	lFlag := flag.Bool("l", false, "hide molgif banner")
	pFlag := flag.Bool("p", false, "render image in png format")
	rFlag := flag.Uint("r", 0, "background color red component")
	gFlag := flag.Uint("g", 0, "background color green component")
	bFlag := flag.Uint("b", 0, "background color blue component")
	aFlag := flag.Float64("a", 0.4, "atom size")
	dFlag := flag.Float64("d", 0.2, "bond size")
	flag.Parse()
	if *eFlag != "" {
		f, err := os.Create(*eFlag)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *wFlag < 1 || *hFlag < 1 {
		log.Fatal("image width and height must be positive")
	}
	if *tFlag < 1 {
		log.Fatal("loop time must be positive")
	}
	if *rFlag > 255 || *gFlag > 255 || *bFlag > 255 {
		log.Fatal("color component must be in the [0, 255] range")
	}
	if *aFlag < 0 {
		log.Fatal("atom size must be positive")
	}
	if *dFlag < 0 {
		log.Fatal("bond size must be positive")
	}
	if *aFlag < *dFlag {
		*dFlag = *aFlag
	}
	inp := flag.Arg(0)
	if inp == "" {
		log.Fatal("specify input file name")
	}
	mol, err := NewMolecule(inp)
	if err != nil {
		log.Fatal(err)
	}
	if *oFlag == "" {
		suf := ".gif"
		if *pFlag {
			suf = ".png"
		}
		base := (inp)[:len(inp)-len(path.Ext(inp))]
		*oFlag = base + suf
	}
	if !*xFlag && !*yFlag && !*zFlag && !*XFlag && !*YFlag && !*ZFlag {
		*yFlag = true
	}
	f, err := os.Create(*oFlag)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	sc := NewScene(mol, *wFlag, *hFlag)
	sc.banner = !*lFlag
	sc.bg = color.RGBA{uint8(*rFlag), uint8(*gFlag), uint8(*bFlag), 255}
	sc.atomSize = float32(*aFlag)
	sc.bondSize = float32(*dFlag)
	sc.UpdateGeometry()
	if *pFlag {
		m := sc.Render()
		if err = png.Encode(f, m); err != nil {
			log.Fatal(err)
		}
	} else {
		var rotvec [3]float32
		if *xFlag {
			rotvec[0] = 1
		}
		if *yFlag {
			rotvec[1] = 1
		}
		if *zFlag {
			rotvec[2] = 1
		}
		if *XFlag {
			rotvec[0] = -1
		}
		if *YFlag {
			rotvec[1] = -1
		}
		if *ZFlag {
			rotvec[2] = -1
		}
		fmt.Print(f.Name())
		g := RenderAll(sc, *tFlag, rotvec)
		if err = gif.EncodeAll(f, g); err != nil {
			log.Fatal(err)
		}
	}
}

var Elements map[string]color.RGBA = map[string]color.RGBA{
	"H":  color.RGBA{255, 255, 255, 255},
	"He": color.RGBA{217, 255, 255, 255},
	"Li": color.RGBA{205, 126, 255, 255},
	"Be": color.RGBA{197, 255, 0, 255},
	"B":  color.RGBA{255, 183, 183, 255},
	"C":  color.RGBA{146, 146, 146, 255},
	"N":  color.RGBA{143, 143, 255, 255},
	"O":  color.RGBA{240, 0, 0, 255},
	"F":  color.RGBA{179, 255, 255, 255},
	"Ne": color.RGBA{175, 227, 244, 255},
	"Na": color.RGBA{170, 94, 242, 255},
	"Mg": color.RGBA{137, 255, 0, 255},
	"Al": color.RGBA{210, 165, 165, 255},
	"Si": color.RGBA{129, 154, 154, 255},
	"P":  color.RGBA{255, 128, 0, 255},
	"S":  color.RGBA{255, 200, 50, 255},
	"Cl": color.RGBA{32, 240, 32, 255},
	"Ar": color.RGBA{129, 209, 228, 255},
	"K":  color.RGBA{143, 65, 211, 255},
	"Ca": color.RGBA{61, 255, 0, 255},
	"Sc": color.RGBA{230, 230, 228, 255},
	"Ti": color.RGBA{192, 195, 198, 255},
	"V":  color.RGBA{167, 165, 172, 255},
	"Cr": color.RGBA{139, 153, 198, 255},
	"Mn": color.RGBA{156, 123, 198, 255},
	"Fe": color.RGBA{129, 123, 198, 255},
	"Co": color.RGBA{112, 123, 195, 255},
	"Ni": color.RGBA{93, 123, 195, 255},
	"Cu": color.RGBA{255, 123, 98, 255},
	"Zn": color.RGBA{124, 129, 175, 255},
	"Ga": color.RGBA{195, 146, 145, 255},
	"Ge": color.RGBA{102, 146, 146, 255},
	"As": color.RGBA{190, 129, 227, 255},
	"Se": color.RGBA{255, 162, 0, 255},
	"Br": color.RGBA{165, 42, 42, 255},
	"Kr": color.RGBA{93, 186, 209, 255},
	"Rb": color.RGBA{113, 46, 178, 255},
	"Sr": color.RGBA{0, 254, 0, 255},
	"Y":  color.RGBA{150, 253, 255, 255},
	"Zr": color.RGBA{150, 225, 225, 255},
	"Nb": color.RGBA{116, 195, 203, 255},
	"Mo": color.RGBA{85, 181, 183, 255},
	"Tc": color.RGBA{60, 159, 168, 255},
	"Ru": color.RGBA{35, 142, 151, 255},
	"Rh": color.RGBA{11, 124, 140, 255},
	"Pd": color.RGBA{0, 104, 134, 255},
	"Ag": color.RGBA{153, 198, 255, 255},
	"Cd": color.RGBA{255, 216, 145, 255},
	"In": color.RGBA{167, 118, 115, 255},
	"Sn": color.RGBA{102, 129, 129, 255},
	"Sb": color.RGBA{159, 101, 181, 255},
	"Te": color.RGBA{213, 123, 0, 255},
	"I":  color.RGBA{147, 0, 147, 255},
	"Xe": color.RGBA{66, 159, 176, 255},
	"Cs": color.RGBA{87, 25, 143, 255},
	"Ba": color.RGBA{0, 202, 0, 255},
	"La": color.RGBA{112, 222, 255, 255},
	"Ce": color.RGBA{255, 255, 200, 255},
	"Pr": color.RGBA{217, 255, 200, 255},
	"Nd": color.RGBA{198, 255, 200, 255},
	"Pm": color.RGBA{164, 255, 200, 255},
	"Sm": color.RGBA{146, 255, 200, 255},
	"Eu": color.RGBA{99, 255, 200, 255},
	"Gd": color.RGBA{71, 255, 200, 255},
	"Tb": color.RGBA{50, 255, 200, 255},
	"Dy": color.RGBA{31, 255, 183, 255},
	"Ho": color.RGBA{0, 254, 157, 255},
	"Er": color.RGBA{0, 230, 118, 255},
	"Tm": color.RGBA{0, 210, 83, 255},
	"Yb": color.RGBA{0, 191, 57, 255},
	"Lu": color.RGBA{0, 172, 35, 255},
	"Hf": color.RGBA{77, 194, 255, 255},
	"Ta": color.RGBA{77, 167, 255, 255},
	"W":  color.RGBA{39, 148, 214, 255},
	"Re": color.RGBA{39, 126, 172, 255},
	"Os": color.RGBA{39, 104, 151, 255},
	"Ir": color.RGBA{24, 85, 135, 255},
	"Pt": color.RGBA{24, 91, 145, 255},
	"Au": color.RGBA{255, 209, 36, 255},
	"Hg": color.RGBA{181, 181, 195, 255},
	"Tl": color.RGBA{167, 85, 77, 255},
	"Pb": color.RGBA{87, 90, 96, 255},
	"Bi": color.RGBA{159, 79, 181, 255},
	"Po": color.RGBA{172, 93, 0, 255},
	"At": color.RGBA{118, 79, 69, 255},
	"Rn": color.RGBA{66, 132, 151, 255},
	"Fr": color.RGBA{66, 0, 102, 255},
	"Ra": color.RGBA{0, 123, 0, 255},
	"Ac": color.RGBA{113, 170, 252, 255},
	"Th": color.RGBA{0, 186, 255, 255},
	"Pa": color.RGBA{0, 160, 255, 255},
	"U":  color.RGBA{0, 145, 255, 255},
	"Np": color.RGBA{0, 128, 242, 255},
	"Pu": color.RGBA{0, 106, 242, 255},
	"Am": color.RGBA{85, 91, 242, 255},
	"Cm": color.RGBA{120, 91, 227, 255},
	"Bk": color.RGBA{137, 79, 227, 255},
	"Cf": color.RGBA{161, 55, 213, 255},
	"Es": color.RGBA{179, 31, 213, 255},
	"Fm": color.RGBA{179, 31, 186, 255},
	"Md": color.RGBA{179, 13, 167, 255},
	"No": color.RGBA{189, 13, 135, 255},
	"Lr": color.RGBA{201, 0, 102, 255},
}
