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
	"strings"
	"sync"
)

type Mat struct {
	xx, xy, xz float32
	yx, yy, yz float32
	zx, zy, zz float32
}

func MatVec(m Mat, v Vec) Vec {
	return Vec{
		m.xx*v.x + m.xy*v.y + m.xz*v.z,
		m.yx*v.x + m.yy*v.y + m.yz*v.z,
		m.zx*v.x + m.zy*v.y + m.zz*v.z,
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
	origin, dir Vec // dir is normalized
}

type Shape interface {
	Intersect(Ray) (float32, Vec, Vec)
	Center() Vec
	Material() Material
}

type Sphere struct {
	pos      Vec
	radius   float32
	material Material
}

func (s *Sphere) Intersect(ray Ray) (float32, Vec, Vec) {
	l := VecSub(s.pos, ray.origin)
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
	p.Add(ray.origin)
	n := p
	n.Sub(s.pos)
	n.Normalize()
	return t, p, n
}

func (s *Sphere) Center() Vec {
	return s.pos
}

func (s *Sphere) Material() Material {
	return s.material
}

type Cylinder struct {
	pos            Vec
	dir            Vec
	height, radius float32
	material       Material
}

func NewCylinder(a, b Vec) Cylinder {
	return Cylinder{}
}

func (c *Cylinder) Intersect(ray Ray) (float32, Vec, Vec) {
	return math.MaxFloat32, Vec{}, Vec{}
}

func (c *Cylinder) Center() Vec {
	return c.pos
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
	dx := float32(x-v.width/2) / float32(v.width)
	dy := float32(y-v.height/2) / float32(v.width)
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

type Material struct {
	color color.RGBA
}

type Element struct {
	material Material
	radius   float32
}

type Molecule struct {
	atoms []*Atom
}

func (m *Molecule) MoveToOrigin() {
	var c Vec
	for _, a := range m.atoms {
		c.Add(a.pos)
	}
	c.Scale(1.0 / float32(len(m.atoms)))
	for _, a := range m.atoms {
		a.pos.Sub(c)
	}
}

func (m *Molecule) MakeBonds() []*Bond {
	const thresh = 1.6
	var bonds []*Bond
	for _, a := range m.atoms {
		for _, b := range m.atoms {
			pa := a.pos
			pb := b.pos
			if d := VecSub(pa, pb); d.LenSq() < thresh*thresh {
				bnd := Bond{a, b}
				bonds = append(bonds, &bnd)
			}
		}
	}
	return bonds
}

func (m *Molecule) Rotate(angle float32) {
	r := MatRotY(angle)
	for _, a := range m.atoms {
		a.pos = MatVec(r, a.pos)
	}
}

func (m *Molecule) Geometry() []Shape {
	var g []Shape
	for _, a := range m.atoms {
		e := elements[a.name]
		s := Sphere{a.pos, e.radius, e.material}
		g = append(g, &s)
	}
	bonds := m.MakeBonds()
	for _, b := range bonds {
		s := NewCylinder(b.a.pos, b.b.pos)
		g = append(g, &s)
	}
	return g
}

func NewMolecule(path string) (*Molecule, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m := new(Molecule)
	sc := bufio.NewScanner(f)
	sc.Scan() // skip atom count
	sc.Scan() // skip comment
	for sc.Scan() {
		var v Vec
		var s string
		fmt.Sscanf(sc.Text(), "%s %f %f %f", &s, &v.x, &v.y, &v.z)
		s = strings.Title(s)
		_, ok := elements[s]
		if !ok {
			return nil, fmt.Errorf("unknown element: %s", s)
		}
		atom := Atom{s, v}
		m.atoms = append(m.atoms, &atom)
	}
	if err = sc.Err(); err != nil {
		return nil, err
	}
	if len(m.atoms) == 0 {
		return nil, fmt.Errorf("%s: no atoms found", path)
	}
	m.MoveToOrigin()
	return m, nil
}

type PointLight struct {
	pos Vec
}

type Scene struct {
	shapes []Shape
	lights []PointLight
	view   *View
	bg     color.RGBA
}

func NewScene(shapes []Shape, bg color.RGBA, w, h int) *Scene {
	var r float32 = 0
	for _, s := range shapes {
		c := s.Center()
		l := c.Len()
		if l > r {
			r = l
		}
	}
	s := Scene{
		shapes: shapes,
		view:   NewView(w, h, r+10.0),
		bg:     bg,
	}
	l := PointLight{Vec{10, 10, -10}}
	s.lights = append(s.lights, l)
	return &s
}

func (p *Scene) RenderTile(b image.Rectangle) image.Image {
	img := image.NewRGBA(b)
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			pix := p.bg
			var zmin float32 = math.MaxFloat32
			ray := p.view.NewRay(x, y)
			for _, s := range p.shapes {
				z, v, n := s.Intersect(ray)
				if z < zmin {
					zmin = z
					l := p.lights[0].pos
					l.Sub(v)
					l.Normalize()
					dot := VecDot(l, n)
					if dot < 0.0 {
						dot = 0.0
					}
					pix = s.Material().color
					pix.R = uint8(float32(pix.R) * dot)
					pix.G = uint8(float32(pix.G) * dot)
					pix.B = uint8(float32(pix.B) * dot)
				}
			}
			img.Set(x, y, pix)
		}
	}
	return img
}

func (p *Scene) UpdateGeometry(m *Molecule) {
	p.shapes = m.Geometry()
}

func (p *Scene) Render() image.Image {
	const tileSize = 64
	bounds := image.Rect(0, 0, p.view.width, p.view.height)
	img := image.NewRGBA(bounds)
	np := runtime.NumCPU()
	ntilx := (bounds.Dx() + tileSize - 1) / tileSize
	ntily := (bounds.Dy() + tileSize - 1) / tileSize
	var wg sync.WaitGroup
	in := make(chan image.Rectangle)
	out := make(chan image.Image, ntilx*ntily)
	for i := 0; i < np; i++ {
		wg.Add(1)
		// render frame in parallel
		go func() {
			defer wg.Done()
			for t := range in {
				out <- p.RenderTile(t)
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
	return img
}

func MakePaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	pm := image.NewPaletted(bounds, palette.Plan9)
	draw.FloydSteinberg.Draw(pm, bounds, img, image.ZP)
	return pm
}

func RenderAll(s *Scene, m *Molecule, loopTime int, rx, ry, rz bool) *gif.GIF {
	const FPS = 50
	nframes := loopTime * FPS
	ang := 2.0 * math.Pi / float32(nframes)
	angv := Vec{}
	if rx {
		angv.x = ang
	}
	if ry {
		angv.y = ang
	}
	if rz {
		angv.z = ang
	}
	var g gif.GIF
	for i := 0; i < nframes; i++ {
		m.Rotate(ang)
		s.UpdateGeometry(m)
		img := s.Render()
		g.Image = append(g.Image, MakePaletted(img))
		g.Delay = append(g.Delay, 100/FPS)
	}
	return &g
}

const (
	StyleDefault = iota
	StyleNoBonds
	StyleFatBonds
	StyleBalls
	StyleLast
)

func main() {
	oFlag := flag.String("o", "", "output file name")
	wFlag := flag.Int("w", 300, "output image width")
	hFlag := flag.Int("h", 200, "output image height")
	xFlag := flag.Bool("x", false, "rotate along x axis")
	yFlag := flag.Bool("y", false, "rotate along y axis")
	zFlag := flag.Bool("z", false, "rotate along z axis")
	tFlag := flag.Int("t", 2, "animation loop time in seconds")
	rFlag := flag.Uint("r", 0, "background color red component")
	gFlag := flag.Uint("g", 0, "background color green component")
	bFlag := flag.Uint("b", 0, "background color blue component")
	pFlag := flag.Bool("p", false, "render one frame in png format")
	sFlag := flag.Uint("s", StyleDefault, "rendering style")
	flag.Parse()
	if *wFlag < 1 || *hFlag < 1 {
		log.Fatal("image width and height must be positive")
	}
	if *tFlag < 1 {
		log.Fatal("loop time must be positive")
	}
	if *rFlag > 255 || *gFlag > 255 || *bFlag > 255 {
		log.Fatal("color component must be in the [0, 255] range")
	}
	if *sFlag >= StyleLast {
		*sFlag = StyleDefault
	}
	inp := flag.Arg(0)
	if inp == "" {
		inp = "sample.xyz"
	}
	m, err := NewMolecule(inp)
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
	if !*xFlag && !*yFlag && !*zFlag {
		*yFlag = true
	}
	f, err := os.Create(*oFlag)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	bg := color.RGBA{uint8(*rFlag), uint8(*gFlag), uint8(*bFlag), 255}
	r := NewScene(m.Geometry(), bg, *wFlag, *hFlag)
	if *pFlag {
		g := r.Render()
		if err = png.Encode(f, g); err != nil {
			log.Fatal(err)
		}
	} else {
		g := RenderAll(r, m, *tFlag, *xFlag, *yFlag, *zFlag)
		if err = gif.EncodeAll(f, g); err != nil {
			log.Fatal(err)
		}
	}
}

var elements map[string]Element = map[string]Element{
	"H":  {Material{color: color.RGBA{255, 255, 255, 255}}, 0.5},
	"He": {Material{color: color.RGBA{217, 255, 255, 255}}, 0.5},
	"Li": {Material{color: color.RGBA{205, 126, 255, 255}}, 0.5},
	"Be": {Material{color: color.RGBA{197, 255, 0, 255}}, 0.5},
	"B":  {Material{color: color.RGBA{255, 183, 183, 255}}, 0.5},
	"C":  {Material{color: color.RGBA{146, 146, 146, 255}}, 0.5},
	"N":  {Material{color: color.RGBA{143, 143, 255, 255}}, 0.5},
	"O":  {Material{color: color.RGBA{240, 0, 0, 255}}, 0.5},
	"F":  {Material{color: color.RGBA{179, 255, 255, 255}}, 0.5},
	"Ne": {Material{color: color.RGBA{175, 227, 244, 255}}, 0.5},
	"Na": {Material{color: color.RGBA{170, 94, 242, 255}}, 0.5},
	"Mg": {Material{color: color.RGBA{137, 255, 0, 255}}, 0.5},
	"Al": {Material{color: color.RGBA{210, 165, 165, 255}}, 0.5},
	"Si": {Material{color: color.RGBA{129, 154, 154, 255}}, 0.5},
	"P":  {Material{color: color.RGBA{255, 128, 0, 255}}, 0.5},
	"S":  {Material{color: color.RGBA{255, 200, 50, 255}}, 0.5},
	"Cl": {Material{color: color.RGBA{32, 240, 32, 255}}, 0.5},
	"Ar": {Material{color: color.RGBA{129, 209, 228, 255}}, 0.5},
	"K":  {Material{color: color.RGBA{143, 65, 211, 255}}, 0.5},
	"Ca": {Material{color: color.RGBA{61, 255, 0, 255}}, 0.5},
	"Sc": {Material{color: color.RGBA{230, 230, 228, 255}}, 0.5},
	"Ti": {Material{color: color.RGBA{192, 195, 198, 255}}, 0.5},
	"V":  {Material{color: color.RGBA{167, 165, 172, 255}}, 0.5},
	"Cr": {Material{color: color.RGBA{139, 153, 198, 255}}, 0.5},
	"Mn": {Material{color: color.RGBA{156, 123, 198, 255}}, 0.5},
	"Fe": {Material{color: color.RGBA{129, 123, 198, 255}}, 0.5},
	"Co": {Material{color: color.RGBA{112, 123, 195, 255}}, 0.5},
	"Ni": {Material{color: color.RGBA{93, 123, 195, 255}}, 0.5},
	"Cu": {Material{color: color.RGBA{255, 123, 98, 255}}, 0.5},
	"Zn": {Material{color: color.RGBA{124, 129, 175, 255}}, 0.5},
	"Ga": {Material{color: color.RGBA{195, 146, 145, 255}}, 0.5},
	"Ge": {Material{color: color.RGBA{102, 146, 146, 255}}, 0.5},
	"As": {Material{color: color.RGBA{190, 129, 227, 255}}, 0.5},
	"Se": {Material{color: color.RGBA{255, 162, 0, 255}}, 0.5},
	"Br": {Material{color: color.RGBA{165, 42, 42, 255}}, 0.5},
	"Kr": {Material{color: color.RGBA{93, 186, 209, 255}}, 0.5},
	"Rb": {Material{color: color.RGBA{113, 46, 178, 255}}, 0.5},
	"Sr": {Material{color: color.RGBA{0, 254, 0, 255}}, 0.5},
	"Y":  {Material{color: color.RGBA{150, 253, 255, 255}}, 0.5},
	"Zr": {Material{color: color.RGBA{150, 225, 225, 255}}, 0.5},
	"Nb": {Material{color: color.RGBA{116, 195, 203, 255}}, 0.5},
	"Mo": {Material{color: color.RGBA{85, 181, 183, 255}}, 0.5},
	"Tc": {Material{color: color.RGBA{60, 159, 168, 255}}, 0.5},
	"Ru": {Material{color: color.RGBA{35, 142, 151, 255}}, 0.5},
	"Rh": {Material{color: color.RGBA{11, 124, 140, 255}}, 0.5},
	"Pd": {Material{color: color.RGBA{0, 104, 134, 255}}, 0.5},
	"Ag": {Material{color: color.RGBA{153, 198, 255, 255}}, 0.5},
	"Cd": {Material{color: color.RGBA{255, 216, 145, 255}}, 0.5},
	"In": {Material{color: color.RGBA{167, 118, 115, 255}}, 0.5},
	"Sn": {Material{color: color.RGBA{102, 129, 129, 255}}, 0.5},
	"Sb": {Material{color: color.RGBA{159, 101, 181, 255}}, 0.5},
	"Te": {Material{color: color.RGBA{213, 123, 0, 255}}, 0.5},
	"I":  {Material{color: color.RGBA{147, 0, 147, 255}}, 0.5},
	"Xe": {Material{color: color.RGBA{66, 159, 176, 255}}, 0.5},
	"Cs": {Material{color: color.RGBA{87, 25, 143, 255}}, 0.5},
	"Ba": {Material{color: color.RGBA{0, 202, 0, 255}}, 0.5},
	"La": {Material{color: color.RGBA{112, 222, 255, 255}}, 0.5},
	"Ce": {Material{color: color.RGBA{255, 255, 200, 255}}, 0.5},
	"Pr": {Material{color: color.RGBA{217, 255, 200, 255}}, 0.5},
	"Nd": {Material{color: color.RGBA{198, 255, 200, 255}}, 0.5},
	"Pm": {Material{color: color.RGBA{164, 255, 200, 255}}, 0.5},
	"Sm": {Material{color: color.RGBA{146, 255, 200, 255}}, 0.5},
	"Eu": {Material{color: color.RGBA{99, 255, 200, 255}}, 0.5},
	"Gd": {Material{color: color.RGBA{71, 255, 200, 255}}, 0.5},
	"Tb": {Material{color: color.RGBA{50, 255, 200, 255}}, 0.5},
	"Dy": {Material{color: color.RGBA{31, 255, 183, 255}}, 0.5},
	"Ho": {Material{color: color.RGBA{0, 254, 157, 255}}, 0.5},
	"Er": {Material{color: color.RGBA{0, 230, 118, 255}}, 0.5},
	"Tm": {Material{color: color.RGBA{0, 210, 83, 255}}, 0.5},
	"Yb": {Material{color: color.RGBA{0, 191, 57, 255}}, 0.5},
	"Lu": {Material{color: color.RGBA{0, 172, 35, 255}}, 0.5},
	"Hf": {Material{color: color.RGBA{77, 194, 255, 255}}, 0.5},
	"Ta": {Material{color: color.RGBA{77, 167, 255, 255}}, 0.5},
	"W":  {Material{color: color.RGBA{39, 148, 214, 255}}, 0.5},
	"Re": {Material{color: color.RGBA{39, 126, 172, 255}}, 0.5},
	"Os": {Material{color: color.RGBA{39, 104, 151, 255}}, 0.5},
	"Ir": {Material{color: color.RGBA{24, 85, 135, 255}}, 0.5},
	"Pt": {Material{color: color.RGBA{24, 91, 145, 255}}, 0.5},
	"Au": {Material{color: color.RGBA{255, 209, 36, 255}}, 0.5},
	"Hg": {Material{color: color.RGBA{181, 181, 195, 255}}, 0.5},
	"Tl": {Material{color: color.RGBA{167, 85, 77, 255}}, 0.5},
	"Pb": {Material{color: color.RGBA{87, 90, 96, 255}}, 0.5},
	"Bi": {Material{color: color.RGBA{159, 79, 181, 255}}, 0.5},
	"Po": {Material{color: color.RGBA{172, 93, 0, 255}}, 0.5},
	"At": {Material{color: color.RGBA{118, 79, 69, 255}}, 0.5},
	"Rn": {Material{color: color.RGBA{66, 132, 151, 255}}, 0.5},
	"Fr": {Material{color: color.RGBA{66, 0, 102, 255}}, 0.5},
	"Ra": {Material{color: color.RGBA{0, 123, 0, 255}}, 0.5},
	"Ac": {Material{color: color.RGBA{113, 170, 252, 255}}, 0.5},
	"Th": {Material{color: color.RGBA{0, 186, 255, 255}}, 0.5},
	"Pa": {Material{color: color.RGBA{0, 160, 255, 255}}, 0.5},
	"U":  {Material{color: color.RGBA{0, 145, 255, 255}}, 0.5},
	"Np": {Material{color: color.RGBA{0, 128, 242, 255}}, 0.5},
	"Pu": {Material{color: color.RGBA{0, 106, 242, 255}}, 0.5},
	"Am": {Material{color: color.RGBA{85, 91, 242, 255}}, 0.5},
	"Cm": {Material{color: color.RGBA{120, 91, 227, 255}}, 0.5},
	"Bk": {Material{color: color.RGBA{137, 79, 227, 255}}, 0.5},
	"Cf": {Material{color: color.RGBA{161, 55, 213, 255}}, 0.5},
	"Es": {Material{color: color.RGBA{179, 31, 213, 255}}, 0.5},
	"Fm": {Material{color: color.RGBA{179, 31, 186, 255}}, 0.5},
	"Md": {Material{color: color.RGBA{179, 13, 167, 255}}, 0.5},
	"No": {Material{color: color.RGBA{189, 13, 135, 255}}, 0.5},
	"Lr": {Material{color: color.RGBA{201, 0, 102, 255}}, 0.5},
}
