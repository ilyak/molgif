package main

import (
	"bufio"
	"flag"
	"fmt"
	"image/color"
	"math"
	"os"
)

type Vec struct {
	x, y, z float32
}

func (v *Vec) LenSq() float32 {
	return v.x*v.x + v.y*v.y + v.z*v.z
}

func (v *Vec) Len() float32 {
	return float32(math.Sqrt(float64(v.LenSq())))
}

func (v *Vec) Normalize() {
	l := v.Len()
	v.x /= l
	v.y /= l
	v.z /= l
}

type Ray struct {
	dir, origin Vec
}

type Shape interface {
	Intersect(Ray) (bool, pt Vec, n Vec)
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
	center   Vec
	dir      Vec
	height   float32
	material Material
}

func (c *Cylinder) Intersect(ray Ray) (bool, Vec, Vec) {
	return false, Vec{}, Vec{}
}

type Camera struct {
	pos, dir Vec
}

type Material struct {
	color color.Color
}

type Atom struct {
	name   string
	sphere Sphere
}

type Bond struct {
	a, b     *Atom
	cylinder Cylinder
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
	atoms []Atom
	bonds []Bond
}

func (m *Molecule) MakeBonds() {
}

func (m *Molecule) Intersect(ray Ray) (bool, Vec, Vec) {
	return false, Vec{}, Vec{}
}

func NewMolecule(path string) (error, *Molecule) {
	f, err := os.Open(path)
	if err != nil {
		return err, nil
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
			return fmt.Errorf("unknown element: %s", s), nil
		}
		atom := Atom{s, Sphere{v, e.radius, e.material}}
		m.atoms = append(m.atoms, atom)
	}
	if err = sc.Err(); err != nil {
		return err, nil
	}
	if len(m.atoms) == 0 {
		return fmt.Errorf("%s: no atoms found", path), nil
	}
	m.MakeBonds()
	return nil, m
}

type Color color.RGBA

func (c *Color) Set(val string) error {
	_, err := fmt.Sscanf(val, "{%d,%d,%d}", &c.R, &c.G, &c.B)
	return err
}

func (c Color) String() string {
	return fmt.Sprintf("{%d,%d,%d}", c.R, c.G, c.B)
}

var iFlag *string = flag.String("i", "input.xyz", "input file")
var oFlag *string = flag.String("o", "output.gif", "output file")
var wFlag *int = flag.Int("w", 300, "output image width")
var hFlag *int = flag.Int("h", 200, "output image height")
var xFlag *int = flag.Int("x", 0, "rotation speed along x axis")
var yFlag *int = flag.Int("y", 100, "rotation speed along y axis")
var zFlag *int = flag.Int("z", 0, "rotation speed along z axis")
var bFlag Color

func main() {
	flag.Var(&bFlag, "b", "background color")
	flag.Parse()
	err, m := NewMolecule(*iFlag)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%v\n", m)
	fmt.Println(bFlag)
}
