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
	GetCenter() Vec
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

func (s *Sphere) GetCenter() Vec {
	return s.pos
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

func (c *Cylinder) GetCenter() Vec {
	return c.pos
}

type View struct {
	width, height        int
	pos, look, right, up Vec
	viewdist             float32
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

//func (v *View) Rotate(dx, dy, dz float32) {
//}

//func (v *View) Advance(angv Vec) {
//}

func (v *View) Orient(phi, theta, psi float32) {
}

func (v *View) NewRay(x, y int) Ray {
	dx := float32(x - v.width/2)
	dy := float32(y - v.height/2)
	dir := v.look
	dir.Add(VecScale(v.right, dx))
	dir.Add(VecScale(v.up, dy))
	return Ray{v.pos, dir}
}

type Material struct {
	color color.Color
}

type Atom struct {
	name  string
	shape Sphere
}

type Bond struct {
	a, b  *Atom
	shape Cylinder
}

type Element struct {
	material Material
	radius   float32
}

var elements map[string]Element = map[string]Element{
	"H":  {Material{color: color.RGBA{255, 255, 255, 255}}, 1.0},
	"He": {Material{color: color.RGBA{217, 255, 255, 255}}, 1.0},
	"Li": {Material{color: color.RGBA{205, 126, 255, 255}}, 1.0},
	"Be": {Material{color: color.RGBA{197, 255, 0, 255}}, 1.0},
	"B":  {Material{color: color.RGBA{255, 183, 183, 255}}, 1.0},
	"C":  {Material{color: color.RGBA{146, 146, 146, 255}}, 1.0},
	"N":  {Material{color: color.RGBA{143, 143, 255, 255}}, 1.0},
	"O":  {Material{color: color.RGBA{240, 0, 0, 255}}, 1.0},
	"F":  {Material{color: color.RGBA{179, 255, 255, 255}}, 1.0},
	"Ne": {Material{color: color.RGBA{175, 227, 244, 255}}, 1.0},
	"Na": {Material{color: color.RGBA{170, 94, 242, 255}}, 1.0},
	"Mg": {Material{color: color.RGBA{137, 255, 0, 255}}, 1.0},
	"Al": {Material{color: color.RGBA{210, 165, 165, 255}}, 1.0},
	"Si": {Material{color: color.RGBA{129, 154, 154, 255}}, 1.0},
	"P":  {Material{color: color.RGBA{255, 128, 0, 255}}, 1.0},
	"S":  {Material{color: color.RGBA{255, 200, 50, 255}}, 1.0},
	"Cl": {Material{color: color.RGBA{32, 240, 32, 255}}, 1.0},
	"Ar": {Material{color: color.RGBA{129, 209, 228, 255}}, 1.0},
	"K":  {Material{color: color.RGBA{143, 65, 211, 255}}, 1.0},
	"Ca": {Material{color: color.RGBA{61, 255, 0, 255}}, 1.0},
	"Sc": {Material{color: color.RGBA{230, 230, 228, 255}}, 1.0},
	"Ti": {Material{color: color.RGBA{192, 195, 198, 255}}, 1.0},
	"V":  {Material{color: color.RGBA{167, 165, 172, 255}}, 1.0},
	"Cr": {Material{color: color.RGBA{139, 153, 198, 255}}, 1.0},
	"Mn": {Material{color: color.RGBA{156, 123, 198, 255}}, 1.0},
	"Fe": {Material{color: color.RGBA{129, 123, 198, 255}}, 1.0},
	"Co": {Material{color: color.RGBA{112, 123, 195, 255}}, 1.0},
	"Ni": {Material{color: color.RGBA{93, 123, 195, 255}}, 1.0},
	"Cu": {Material{color: color.RGBA{255, 123, 98, 255}}, 1.0},
	"Zn": {Material{color: color.RGBA{124, 129, 175, 255}}, 1.0},
	"Ga": {Material{color: color.RGBA{195, 146, 145, 255}}, 1.0},
	"Ge": {Material{color: color.RGBA{102, 146, 146, 255}}, 1.0},
	"As": {Material{color: color.RGBA{190, 129, 227, 255}}, 1.0},
	"Se": {Material{color: color.RGBA{255, 162, 0, 255}}, 1.0},
	"Br": {Material{color: color.RGBA{165, 42, 42, 255}}, 1.0},
	"Kr": {Material{color: color.RGBA{93, 186, 209, 255}}, 1.0},
	"Rb": {Material{color: color.RGBA{113, 46, 178, 255}}, 1.0},
	"Sr": {Material{color: color.RGBA{0, 254, 0, 255}}, 1.0},
	"Y":  {Material{color: color.RGBA{150, 253, 255, 255}}, 1.0},
	"Zr": {Material{color: color.RGBA{150, 225, 225, 255}}, 1.0},
	"Nb": {Material{color: color.RGBA{116, 195, 203, 255}}, 1.0},
	"Mo": {Material{color: color.RGBA{85, 181, 183, 255}}, 1.0},
	"Tc": {Material{color: color.RGBA{60, 159, 168, 255}}, 1.0},
	"Ru": {Material{color: color.RGBA{35, 142, 151, 255}}, 1.0},
	"Rh": {Material{color: color.RGBA{11, 124, 140, 255}}, 1.0},
	"Pd": {Material{color: color.RGBA{0, 104, 134, 255}}, 1.0},
	"Ag": {Material{color: color.RGBA{153, 198, 255, 255}}, 1.0},
	"Cd": {Material{color: color.RGBA{255, 216, 145, 255}}, 1.0},
	"In": {Material{color: color.RGBA{167, 118, 115, 255}}, 1.0},
	"Sn": {Material{color: color.RGBA{102, 129, 129, 255}}, 1.0},
	"Sb": {Material{color: color.RGBA{159, 101, 181, 255}}, 1.0},
	"Te": {Material{color: color.RGBA{213, 123, 0, 255}}, 1.0},
	"I":  {Material{color: color.RGBA{147, 0, 147, 255}}, 1.0},
	"Xe": {Material{color: color.RGBA{66, 159, 176, 255}}, 1.0},
	"Cs": {Material{color: color.RGBA{87, 25, 143, 255}}, 1.0},
	"Ba": {Material{color: color.RGBA{0, 202, 0, 255}}, 1.0},
	"La": {Material{color: color.RGBA{112, 222, 255, 255}}, 1.0},
	"Ce": {Material{color: color.RGBA{255, 255, 200, 255}}, 1.0},
	"Pr": {Material{color: color.RGBA{217, 255, 200, 255}}, 1.0},
	"Nd": {Material{color: color.RGBA{198, 255, 200, 255}}, 1.0},
	"Pm": {Material{color: color.RGBA{164, 255, 200, 255}}, 1.0},
	"Sm": {Material{color: color.RGBA{146, 255, 200, 255}}, 1.0},
	"Eu": {Material{color: color.RGBA{99, 255, 200, 255}}, 1.0},
	"Gd": {Material{color: color.RGBA{71, 255, 200, 255}}, 1.0},
	"Tb": {Material{color: color.RGBA{50, 255, 200, 255}}, 1.0},
	"Dy": {Material{color: color.RGBA{31, 255, 183, 255}}, 1.0},
	"Ho": {Material{color: color.RGBA{0, 254, 157, 255}}, 1.0},
	"Er": {Material{color: color.RGBA{0, 230, 118, 255}}, 1.0},
	"Tm": {Material{color: color.RGBA{0, 210, 83, 255}}, 1.0},
	"Yb": {Material{color: color.RGBA{0, 191, 57, 255}}, 1.0},
	"Lu": {Material{color: color.RGBA{0, 172, 35, 255}}, 1.0},
	"Hf": {Material{color: color.RGBA{77, 194, 255, 255}}, 1.0},
	"Ta": {Material{color: color.RGBA{77, 167, 255, 255}}, 1.0},
	"W":  {Material{color: color.RGBA{39, 148, 214, 255}}, 1.0},
	"Re": {Material{color: color.RGBA{39, 126, 172, 255}}, 1.0},
	"Os": {Material{color: color.RGBA{39, 104, 151, 255}}, 1.0},
	"Ir": {Material{color: color.RGBA{24, 85, 135, 255}}, 1.0},
	"Pt": {Material{color: color.RGBA{24, 91, 145, 255}}, 1.0},
	"Au": {Material{color: color.RGBA{255, 209, 36, 255}}, 1.0},
	"Hg": {Material{color: color.RGBA{181, 181, 195, 255}}, 1.0},
	"Tl": {Material{color: color.RGBA{167, 85, 77, 255}}, 1.0},
	"Pb": {Material{color: color.RGBA{87, 90, 96, 255}}, 1.0},
	"Bi": {Material{color: color.RGBA{159, 79, 181, 255}}, 1.0},
	"Po": {Material{color: color.RGBA{172, 93, 0, 255}}, 1.0},
	"At": {Material{color: color.RGBA{118, 79, 69, 255}}, 1.0},
	"Rn": {Material{color: color.RGBA{66, 132, 151, 255}}, 1.0},
	"Fr": {Material{color: color.RGBA{66, 0, 102, 255}}, 1.0},
	"Ra": {Material{color: color.RGBA{0, 123, 0, 255}}, 1.0},
	"Ac": {Material{color: color.RGBA{113, 170, 252, 255}}, 1.0},
	"Th": {Material{color: color.RGBA{0, 186, 255, 255}}, 1.0},
	"Pa": {Material{color: color.RGBA{0, 160, 255, 255}}, 1.0},
	"U":  {Material{color: color.RGBA{0, 145, 255, 255}}, 1.0},
	"Np": {Material{color: color.RGBA{0, 128, 242, 255}}, 1.0},
	"Pu": {Material{color: color.RGBA{0, 106, 242, 255}}, 1.0},
	"Am": {Material{color: color.RGBA{85, 91, 242, 255}}, 1.0},
	"Cm": {Material{color: color.RGBA{120, 91, 227, 255}}, 1.0},
	"Bk": {Material{color: color.RGBA{137, 79, 227, 255}}, 1.0},
	"Cf": {Material{color: color.RGBA{161, 55, 213, 255}}, 1.0},
	"Es": {Material{color: color.RGBA{179, 31, 213, 255}}, 1.0},
	"Fm": {Material{color: color.RGBA{179, 31, 186, 255}}, 1.0},
	"Md": {Material{color: color.RGBA{179, 13, 167, 255}}, 1.0},
	"No": {Material{color: color.RGBA{189, 13, 135, 255}}, 1.0},
	"Lr": {Material{color: color.RGBA{201, 0, 102, 255}}, 1.0},
}

type Molecule struct {
	atoms []*Atom
	bonds []*Bond
}

func (m *Molecule) Center() {
	var c Vec
	for _, a := range m.atoms {
		c.Add(a.shape.pos)
	}
	c.Scale(1.0 / float32(len(m.atoms)))
	for _, a := range m.atoms {
		a.shape.pos.Sub(c)
	}
}

func (m *Molecule) MakeBonds() {
	const bndist = 1.6
	const bndist2 = bndist * bndist
	for _, a := range m.atoms {
		for _, b := range m.atoms {
			pa := a.shape.pos
			pb := b.shape.pos
			if d := VecSub(pa, pb); d.LenSq() < bndist2 {
				bnd := Bond{a, b, NewCylinder(pa, pb)}
				m.bonds = append(m.bonds, &bnd)
			}
		}
	}
}

func (m *Molecule) Geometry() []Shape {
	var g []Shape
	for _, a := range m.atoms {
		g = append(g, &a.shape)
	}
	for _, b := range m.bonds {
		g = append(g, &b.shape)
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
		e, ok := elements[s]
		if !ok {
			return nil, fmt.Errorf("unknown element: %s", s)
		}
		atom := Atom{s, Sphere{v, e.radius, e.material}}
		m.atoms = append(m.atoms, &atom)
	}
	if err = sc.Err(); err != nil {
		return nil, err
	}
	if len(m.atoms) == 0 {
		return nil, fmt.Errorf("%s: no atoms found", path)
	}
	m.Center()
	m.MakeBonds()
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
	frame  int //XXX remove
}

func NewScene(shapes []Shape, bg color.RGBA, w, h int) *Scene {
	var r float32 = 0
	for _, s := range shapes {
		c := s.GetCenter()
		l := c.LenSq()
		if l > r {
			r = l
		}
	}
	return &Scene{
		shapes: shapes,
		view:   NewView(w, h, r+1.0),
		bg:     bg,
	}
}

//func (p *Scene) Advance(angv Vec) {
//	p.view.Advance(angv)
//}

func (p *Scene) RenderTile(b image.Rectangle) image.Image {
	img := image.NewRGBA(b)
	//blue := color.RGBA{0, 0, uint8(p.frame), 255}
	//red := color.RGBA{uint8(p.frame), 0, 0, 255}
	//if (b.Min.X/64+b.Min.Y/64)%2 == 0 {
	//	draw.Draw(img, b, &image.Uniform{blue}, image.ZP, draw.Src)
	//} else {
	//	draw.Draw(img, b, &image.Uniform{red}, image.ZP, draw.Src)
	//}
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			pix := p.bg
			var zmin float32 = math.MaxFloat32
			ray := p.view.NewRay(x, y)
			for _, s := range p.shapes {
				z, v, n := s.Intersect(ray)
				if z < zmin {
					// compute pixel color
					_, _ = v, n
					pix = color.RGBA{128, 0, 128, 255}
				}
			}
			img.Set(x, y, pix)
		}
	}
	return img
}

func (p *Scene) Render() image.Image {
	const tileSize = 64
	bounds := image.Rect(0, 0, p.view.width, p.view.height)
	img := image.NewRGBA(bounds)
	np := runtime.NumCPU()
	ntilx := bounds.Dx() / tileSize
	if bounds.Dx()%tileSize != 0 {
		ntilx++
	}
	ntily := bounds.Dy() / tileSize
	if bounds.Dy()%tileSize != 0 {
		ntily++
	}
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

func (p *Scene) SetFrame(frame int) {
	p.frame = frame //XXX
	var phi, theta, psi float32 = 0.0, 0.0, 0.0
	p.view.Orient(phi, theta, psi)
}

func MakePaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	pm := image.NewPaletted(bounds, palette.Plan9)
	draw.FloydSteinberg.Draw(pm, bounds, img, image.ZP)
	return pm
}

func RenderAll(s *Scene, loopTime int, rx, ry, rz bool) *gif.GIF {
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
		s.SetFrame(i)
		img := s.Render()
		g.Image = append(g.Image, MakePaletted(img))
		g.Delay = append(g.Delay, 100/FPS)
		//s.Advance(angv)
	}
	return &g
}

func main() {
	oFlag := flag.String("o", "", "output file name")
	wFlag := flag.Int("w", 300, "output image width")
	hFlag := flag.Int("h", 200, "output image height")
	xFlag := flag.Bool("x", false, "rotate along x axis")
	yFlag := flag.Bool("y", false, "rotate along y axis")
	zFlag := flag.Bool("z", false, "rotate along z axis")
	tFlag := flag.Int("t", 1, "animation loop time in seconds")
	rFlag := flag.Uint("r", 0, "background color red component")
	gFlag := flag.Uint("g", 0, "background color green component")
	bFlag := flag.Uint("b", 0, "background color blue component")
	nFlag := flag.Int("n", 0, "render single frame n in png format")
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
		if *nFlag > 0 {
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
	if *nFlag > 0 {
		r.SetFrame(*nFlag - 1)
		g := r.Render()
		if err = png.Encode(f, g); err != nil {
			log.Fatal(err)
		}
	} else {
		g := RenderAll(r, *tFlag, *xFlag, *yFlag, *zFlag)
		if err = gif.EncodeAll(f, g); err != nil {
			log.Fatal(err)
		}
	}
}
