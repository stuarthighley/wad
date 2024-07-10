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

	for i, t := range w.TexturesList {
		fmt.Println("Texture:", i, t.Name, t.Index)
		// createPNGPic(k, t.Picture, w)
		//
	}

	for i, f := range w.FlatsList {
		fmt.Println("Flat:", i, f.Name, f.Index)
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

	createPNGFlat("TEST", w.FlatsList[2], w)

}

func createPNGPic(n string, p *wad.Picture, w *wad.WAD) error {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{int(p.Width), int(p.Height)}
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
	lowRight := image.Point{wad.FlatWidth, wad.FlatHeight}
	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	palette := w.Palettes[0]
	colormap := w.ColorMaps[0]

	// Set color for each pixel.
	for i, b := range flat.Data {
		c := palette[colormap[b]]
		rgb := color.RGBA{c.Red, c.Green, c.Blue, 0xff}
		img.SetRGBA(i%wad.FlatWidth, i/wad.FlatWidth, rgb)
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
