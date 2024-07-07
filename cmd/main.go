package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	"github.com/stuarthighley/wad"
)

func main() {

	log.Println("Starting")

	// Set WAD logger
	wad.SetLogger(log.New(os.Stdout, "", log.LstdFlags))

	// New WAD
	w, err := wad.NewWAD("../DOOM1.WAD")
	if err != nil {
		log.Fatalln(err)
	}

	for k, t := range w.Textures {
		fmt.Println("Texture:", k, t)
		// createPNGPic(k, t.Picture, w)
		//
	}

	// log.Println(w)
	// for k, _ := range w.Pictures {
	// 	log.Println(k)
	// }
	// log.Println(len(w.Pictures))

	// boss := w.Sprites["BOSS"]
	// for _, f := range *boss {
	// 	fmt.Printf("%v\n", f)
	// }

	// l, err := w.ReadLevel("E1M2")
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// pic, err := w.GetPicture("help1")
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// createPNGPic("HELP1", pic, w)

	// // fmt.Printf("%+v %+v", l.BlockMap.Columns*l.BlockMap.Rows, len(l.BlockMap.Blocklists))
	// fmt.Printf("%v", l.Reject[0])
	// for k := range w.PatchPics {
	// 	fmt.Println(k)
	// }

	// for k, s := range w.Sprites {
	// 	createPNGPic(k, s, w)
	// }

}

func createPNGPic(n string, p *wad.Picture, w *wad.WAD) error {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{p.Width, p.Height}
	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	palette := w.Palettes[0]
	colormap := w.ColorMaps[0]

	// Set color for each pixel.
	for x := range p.Columns {
		for y, b := range p.Columns[x] {
			if colormap[b] != w.TransparentIndex {
				c := palette[colormap[b]]
				rgb := color.RGBA{c.Red, c.Green, c.Blue, 0xff}
				img.SetRGBA(x, y, rgb)
			}
		}
	}

	// Encode as PNG.
	f, err := os.Create(fmt.Sprintf("../out/%v.png", n))
	if err != nil {
		log.Printf("%v", err)
		return err
	}
	defer f.Close()
	png.Encode(f, img)
	return nil
}

// createPNGFlat
func createPNGFlat(n string, flat *wad.Flat, w *wad.WAD) error {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{len(flat), len(flat[0])}
	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	palette := w.Palettes[0]
	colormap := w.ColorMaps[0]

	// Set color for each pixel.
	for y := range flat {
		for x, b := range flat[y] {
			c := palette[colormap[b]]
			rgb := color.RGBA{c.Red, c.Green, c.Blue, 0xff}
			img.SetRGBA(x, y, rgb)
		}
	}

	// Encode as PNG.
	f, err := os.Create(fmt.Sprintf("../out/%v.png", n))
	if err != nil {
		log.Printf("%v", err)
		return err
	}
	defer f.Close()
	png.Encode(f, img)
	return nil

}
