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
	"log"
	"math"
	"os"
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

func (v *Vec) Mul(s float32) {
	v.x *= s
	v.y *= s
	v.z *= s
}

func (v *Vec) Div(s float32) {
	v.x /= s
	v.y /= s
	v.z /= s
}

func (v *Vec) Normalize() {
	v.Div(v.Len())
}

type Ray struct {
	dir, origin Vec // dir is normalized
}

type Shape interface {
	Intersect(Ray) (bool, Vec, Vec)
}

type Sphere struct {
	pos      Vec
	radius   float32
	material Material
}

func (s *Sphere) Intersect(ray Ray) (bool, Vec, Vec) {
	return false, Vec{}, Vec{}
}

type Cylinder struct {
	center         Vec
	dir            Vec
	height, radius float32
	material       Material
}

func NewCylinder(a, b Vec) Cylinder {
	return Cylinder{}
}

func (c *Cylinder) Intersect(ray Ray) (bool, Vec, Vec) {
	return false, Vec{}, Vec{}
}

type View struct {
	width, height        int
	pos, look, right, up Vec
	viewdist             float32
}

func NewView(width, height int, radius float32) *View {
	pos := Vec{0, 0, -radius}
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

func (v *View) Rotate(dx, dy, dz float32) {
}

func (v *View) Advance(angxyz [3]float32) {
}

func (v *View) NewRay(x, y int) Ray {
	return Ray{}
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
	"He": {Material{color: color.RGBA{255, 255, 255, 255}}, 1.0},
	"O":  {Material{color: color.RGBA{255, 255, 255, 255}}, 1.0},
	"N":  {Material{color: color.RGBA{255, 255, 255, 255}}, 1.0},
	"P":  {Material{color: color.RGBA{255, 255, 255, 255}}, 1.0},
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
	c.Div(float32(len(m.atoms)))
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

type Renderer struct {
	shapes []Shape
	view   *View
	bg     color.RGBA
}

func NewRenderer(shapes []Shape, bg color.RGBA, w, h int) *Renderer {
	var r float32
	return &Renderer{
		shapes: shapes,
		view:   NewView(w, h, r),
		bg:     bg,
	}
}

func (r *Renderer) Advance(angxyz [3]float32) {
	r.view.Advance(angxyz)
}

func (r *Renderer) Render() image.Image {
	rect := image.Rect(0, 0, r.view.width, r.view.height)
	img := image.NewRGBA(rect)
	return img
}

func MakePaletted(img image.Image) *image.Paletted {
	b := img.Bounds()
	pm := image.NewPaletted(b, palette.Plan9)
	draw.FloydSteinberg.Draw(pm, b, img, image.ZP)
	return pm
}

func RenderAnimation(r *Renderer, loopTime int, rx, ry, rz bool) *gif.GIF {
	const FPS = 50
	nframes := loopTime * FPS
	ang := 2.0 * math.Pi / float32(nframes)
	angxyz := [3]float32{}
	if rx {
		angxyz[0] = ang
	}
	if ry {
		angxyz[1] = ang
	}
	if rz {
		angxyz[2] = ang
	}
	var g gif.GIF
	for i := 0; i < nframes; i++ {
		img := r.Render()
		g.Image = append(g.Image, MakePaletted(img))
		g.Delay = append(g.Delay, 100/FPS)
		r.Advance(angxyz)
	}
	return &g
}

func main() {
	iFlag := flag.String("i", "input.xyz", "input file name")
	oFlag := flag.String("o", "output.gif", "output file name")
	wFlag := flag.Int("w", 300, "output image width")
	hFlag := flag.Int("h", 200, "output image height")
	xFlag := flag.Bool("x", false, "rotate along x axis")
	yFlag := flag.Bool("y", true, "rotate along y axis")
	zFlag := flag.Bool("z", false, "rotate along z axis")
	tFlag := flag.Int("t", 10, "animation loop time in seconds")
	rFlag := flag.Uint("r", 0, "background color red component")
	gFlag := flag.Uint("g", 0, "background color green component")
	bFlag := flag.Uint("b", 0, "background color blue component")
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
	m, err := NewMolecule(*iFlag)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(*oFlag)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	bg := color.RGBA{uint8(*rFlag), uint8(*gFlag), uint8(*bFlag), 255}
	r := NewRenderer(m.Geometry(), bg, *wFlag, *hFlag)
	g := RenderAnimation(r, *tFlag, *xFlag, *yFlag, *zFlag)
	if err = gif.EncodeAll(f, g); err != nil {
		log.Fatal(err)
	}
}
