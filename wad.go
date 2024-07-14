// Package wad provides access to Doom's data archives also known as WAD files.
// The file format is documented in The Unofficial DOOM Specs:
// http://www.gamers.org/dhs/helpdocs/dmsp1666.html

package wad

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"
	"unsafe"

	"golang.org/x/exp/constraints"
)

// WAD is a struct that represents Doom's data archive that contains graphics, sounds, and level
// data. The data is organized as named lumps.
type WAD struct {
	header       *Header
	file         *os.File
	lumpInfos    []LumpInfo
	lumpNums     map[string]int
	Palettes     *Palettes
	ColorMaps    *ColorMaps
	Endoom       *Endoom
	Demos        []Demo
	Dmxgus       *DMXGUS
	patchNames   []string
	Pictures     map[string]*Picture
	Textures     map[string]*Texture
	TexturesList []*Texture
	Flats        map[string]*Flat
	FlatsList    []*Flat
	Sprites      map[string]*Sprite
	// SpriteFrames     map[string]*SpriteFrame
	Sounds           map[string]*Sound
	Scores           map[string]*MusicScore
	levels           map[string]int
	TransparentIndex byte
}

type binHeader struct {
	Magic        [4]byte
	NumLumps     int32
	InfoTableOfs int32
}

type Header struct {
	NumLumps     int
	InfoTableOfs int
}

type binLumpInfo struct {
	Filepos int32
	Size    int32
	Name    String8
}

type LumpInfo struct {
	Name    string
	Filepos int
	Size    int
}

// Sound lumps in the WAD file are stored in the DMX format; which consists of a short header
// followed by raw 8-bit, monaural (PCM) unsigned data, typically at a sampling rate of 11025 Hz,
// although some sounds use 22050 Hz. Each sample is one byte (8 bits).
type Sound struct {
	SampleRate uint
	Samples    []byte
}

type MusicScore struct {
}

type binSide struct {
	XOffset       int16
	YOffset       int16
	UpperTexture  String8
	LowerTexture  String8
	MiddleTexture String8
	SectorNum     int16
}

type Side struct {
	XOffset           float64
	YOffset           float64
	UpperTextureName  string
	LowerTextureName  string
	MiddleTextureName string
	SectorNum         int
	UpperTexture      *Texture
	LowerTexture      *Texture
	MiddleTexture     *Texture
	Sector            *Sector
}

type Vertex struct {
	X, Y float64
}

type binLineSegment struct {
	V1        int16
	V2        int16
	Angle     int16 // Full circle is -32768 to 32767.
	LineNum   int16
	Direction int16 // 0 - same as linedef, 1 - opposite to linedef
	Offset    int16 // Distance along line to start of segment
}

type LineSegment struct {
	V1Num   int
	V2Num   int
	Angle   float64 // Radians
	LineNum int
	IsSideL bool    // false - same as linedef, true - opposite to linedef
	Offset  float64 // Distance along line to start of segment

	V1          Vertex
	V2          Vertex
	Line        *Line
	Side        *Side
	FrontSector *Sector
	BackSector  *Sector
}

type binSubSector struct {
	NumSegments      int16
	StartLineSegment int16
}

type SubSector struct {
	numLineSegments  int
	StartLineSegment int

	LineSegments []LineSegment
	Sector       *Sector
}

type BoundBox struct {
	Top, Bottom, Left, Right float64
}

type BlockBox struct {
	Top, Bottom, Left, Right int
}

type binNode struct {
	X, Y                 int16
	DX, DY               int16
	BBoxR, BBoxL         binBBox
	ChildNumR, ChildNumL int16
}

type Node struct {
	X, Y                 float64
	DX, DY               float64
	BBoxR, BBoxL         BoundBox
	ChildNumR, ChildNumL int
	ChildR, ChildL       BSPMember
	// Node                 [2]*Node
	// SubSector            [2]*SubSector
}

// Return child for side
func (n *Node) Child(side int) BSPMember {
	if side == 0 {
		return n.ChildR
	}
	return n.ChildL
}

// Return bound box for side
func (n *Node) BoundBox(side int) *BoundBox {
	if side == 0 {
		return &n.BBoxR
	}
	return &n.BBoxL
}

type BSPType int

const (
	BSPNode BSPType = iota
	BSPSubSector
)

type BSPMember interface {
	BSPType() BSPType
}

func (s *SubSector) BSPType() BSPType {
	return BSPSubSector
}

func (s *Node) BSPType() BSPType {
	return BSPNode
}

// Sector

type binSector struct {
	FloorHeight    int16
	CeilingHeight  int16
	FloorTexture   String8
	CeilingTexture String8
	LightLevel     int16
	Type           int16
	TagNum         int16
}

type Sector struct {
	Index              int
	FloorHeight        float64
	CeilingHeight      float64
	FloorTextureName   string
	CeilingTextureName string
	LightLevel         int
	Type               SectorType
	TagNum             int

	FloorTexture   *Flat
	CeilingTexture *Flat
	Lines          []*Line
	SoundOrigin    Point    // origin for any sounds played by the sector
	BlockBox       BlockBox // mapblock bounding box for height changes

	User any // User data (Doom will store fields as below)
	// Soundtraversed int      // 0 = untraversed, 1,2 = sndlines -1
	// Soundtarget    *Mobj    // thing that made a sound (or null)
	// Validcount     int      // if == validcount, already checked
	// Thinglist      *Mobj    // Root of mobjs in sector linked list	// TODO - or should it be slice?
	// Specialdata    *Thinker // thinker_t for reversable actions
}

type SectorType int

const (
	TypeNormal          SectorType = iota
	TypeBlinkRandom                // 1  Light  Blink random
	TypeBlink05                    // 2  Light  Blink 0.5 second
	TypeBlink10                    // 3  Light  Blink 1.0 second
	TypeDamage20Blink05            // 4  Both   20% damage per second; light blink 0.5 second
	TypeDamage10                   // 5	 Damage 10% damage per second
	TypeUnused1                    // 6  Unused
	TypeDamage5                    // 7	 Damage 5% damage per second
	TypeOscillate                  // 8	 Light  Oscillates
	TypeSecret                     // 9	 Secret Player entering this sector gets credit for finding a secret
	TypeDoor30                     // 10 Door   30 seconds after level start, ceiling closes like a door
	TypeEnd                        // 11 End    20% damage ps. Level ends when player health drops below 11% & touching floor
	TypeBlink10Sync                // 12 Light  Blink 1.0 second, synchronized
	TypeBlink05Sync                // 13 Light  Blink 0.5 second, synchronized
	TypeDoor300                    // 14 Door   300 seconds after level start, ceiling opens like a door
	TypeUnused2                    // 15 Unused
	TypeDamage20                   // 16 Damage 20% damage per second
	TypeFlickerRandom              // 17 Light  Flickers randomly
)

type Point struct {
	X, Y, Z float64
}

type Playpal struct {
	Palettes [14]Palette
}

type Reject [][]bool

type binBlockMapHeader struct {
	OriginX, OriginY int16
	Columns, Rows    int16
}
type binBlockListOffset int16
type binBlockLineNum uint16

// BlockMap is level data created from axis aligned bounding box of the map, a rectangular array
// of blocks of size ... Used to speed up collision detection by spatial subdivision in 2D.
type BlockMap struct {
	OriginX, OriginY    float64
	NumColumns, NumRows int
	Blocks              []Block
}

type Block struct {
	LineNums []int
	Lines    []*Line
}

// Block returns a pointer to the specified block from the block map
func (b *BlockMap) Block(x, y int) *Block {
	return &b.Blocks[y*b.NumColumns+x]
}

// Music formats
type binMusicHeader struct {
	ID              [4]byte // identifier "MUS" 0x1A
	ScoreLen        uint16  // score length in bytes
	ScoreStart      uint16  // the absolute file position of the score
	PrimaryCount    uint16  // count of primary channels
	SecondaryCount  uint16  // count of secondary channels
	InstrumentCount uint16
	Unused          [2]byte
}

type binMusicInstruments []uint16

// Bits 0-3  Channel number
// Bits 4-6  Event type
// Bit 7     Last (if set, event is followed by time information)
// type binSoundEvent byte

type SoundEvent struct {
	ChannelNum int
	EventType  SoundEventType
	Last       bool // if set, the event is followed by time information
}

type SoundEventType int

const (
	ReleaseNote SoundEventType = iota
	PlayNote
	PitchWheel  // Bender
	SystemEvent // valueless controller
	ChangeController
	Unknown1
	ScoreEnd
	Unknown2
)

type binTextureHeader struct {
	TextureName String8
	Masked      int32
	Width       int16
	Height      int16
	Unused      int32 // ColumnDirectory
	NumPatches  int16
}

// type TextureHeader struct {
// 	TexName         string
// 	Masked          int
// 	Width           int
// 	Height          int
// 	ColumnDirectory int
// 	NumPatches      int
// }

type Texture struct {
	Name          string   // Texture name and index into textures map
	Index         int      // Index into TexturesList
	IsMasked      bool     // flag denoting ???
	Width, Height int      // total width and height of the map texture
	Patches       []Patch  // List of component Patches
	Picture       *Picture // Expanded Picture for convenience
}

type binPatch struct {
	XOffset      int16
	YOffset      int16
	PatchNameIdx int16
	Unused1      int16 // StepDir
	Unused2      int16 // ColorMap
}

type Patch struct {
	XOffset int // horizontal offset of patch relative to upper-left of texture
	YOffset int // vertical offset of patch relative to upper-left of texture
	Picture *Picture
}

// The doom picture (image) format. Sometimes called a patch, but this code considers a patch to
// be a parent entity that makes up part of a texture, and points to a picture
type Picture struct {
	Name                  string // Useful for debugging
	Width, Height         int
	LeftOffset, TopOffset int // Allows soulspheres, weapons and keys to float
	Columns               []Column
}

// Rather than implement column posts, just set column to transparent and fill in post data.
type Column []byte

// NewSize creates a new resized picture
func (p *Picture) NewSize(width, height int) *Picture {
	pic := Picture{
		Name:       p.Name,
		Width:      width,
		Height:     height,
		LeftOffset: p.LeftOffset,
		TopOffset:  p.TopOffset,
		Columns:    make([]Column, width),
	}
	for y := range pic.Columns {
		pic.Columns[y] = make(Column, height)
		for x := range pic.Columns[y] {
			pic.Columns[y][x] = p.Columns[y*p.Width/width][x*p.Height/height]
		}
	}
	return &pic
}

// A flat is an image that is drawn on the floors and ceilings of sectors.
// Flats are very different from wall textures. Flats are a raw collection of pixel values with no
// offset or other dimension information; each flat is a named lump of 4096 bytes representing a
// 64×64 square. The pixel values are converted to actual colors in the same way as for the Doom
// picture format, using the colormap.
// Flats are always drawn aligned to a fixed grid. This ensures that floor and ceiling textures
// flow smoothly from sector to sector. It can also cause problems for level designers, usually
// when placing teleport pads.
// Certain flats are animated to represent water, lava, blood, slime, or other substances.

type Flat struct {
	Name  string // Flat name and index into flats map
	Index int    // Index into flats list
	Data  []byte
}

const FlatWidth, FlatHeight = 64, 64

// Sprites are patches with a special naming convention so they can be recognized by R_InitSprites.
// The base name is NNNNFx or NNNNFxFx, with x indicating the rotation, x = 0, 1-7.
// The sprite and frame specified by a thing_t is range checked at run time.
// A sprite is a patch_t that is assumed to represent a 3D object and may have multiple
// rotations pre drawn.
// Horizontal flipping is used to save space, thus NNNNF2F5 defines a mirrored patch.
// Some sprites will only have one picture used for all views: NNNNF0
type Sprite []SpriteFrame

type SpriteFrame [8]SpriteFrameDir

type SpriteFrameDir struct {
	Picture   *Picture
	IsFlipped bool
}

type binSoundHeader struct {
	Format     uint16
	SampleRate uint16
	Bytes      uint32
	Pad        [16]byte
}

type Level struct {
	Things       []Thing
	Lines        []Line
	Sides        []Side
	Vertexes     []Vertex
	LineSegments []LineSegment
	SubSectors   []SubSector
	Nodes        []Node
	Sectors      []Sector
	Reject       Reject
	BlockMap     BlockMap
	RootNode     *Node
}

type binThing struct {
	X       int16
	Y       int16
	Angle   int16
	Type    int16
	Options int16
}

type Thing struct {
	X, Y            int
	Angle           float64
	Type            int
	Skill1and2      bool
	Skill3          bool
	Skill4and5      bool
	Ambush          bool
	MultiplayerOnly bool
}

type binVertex struct {
	X, Y int16
}

type binBBox struct {
	Top    int16
	Bottom int16
	Left   int16
	Right  int16
}

type WadReject struct {
}

type RGB struct {
	Red, Green, Blue uint8
}

// PLAYPAL lump. A set of color palettes used to set the main graphics colors. The Doom engine can
// only display 256 simultaneous colors, so it performs palette swaps to achieve these effects.
type Palettes [14]Palette

// Each palette in PLAYPAL contains 256 three-ubyte colors totaling 768 bytes (RGB).
type Palette [256]RGB

// The COLORMAP lump contains 34 color maps of indices into the PLAYPAL palette chosen at that time
// through which colors can be remapped for sector lighting, distance fading, and partial screen
// color changes (such as the invulnerability effect).
// The COLORMAP resource is built using the PLAYPAL and consists of a number of tables
type ColorMaps [34]ColorMap

// Each color map is a table 256 bytes long. It is indexed using a pixel value (from 0 to 255) and
// yields a new, brightness-adjusted pixel value. So each color has 34 variations. The appropriate
// variation is found by selecting the appropriate color map.
type ColorMap [256]byte

// ENDOOM consists of 4000 bytes representing an 80x25 text block exactly as stored in VGA video
// memory. Every character is stored as two bytes: the first byte is simply the character's 8-bit
// extended ASCII value; the second byte gives color information.
// The second byte is broken into three pieces. Bits 0-3 give the foreground color, 4-6 give the
// background color, and bit 7 is a 'blink' flag. The colors are standard DOS text-mode colors.
type Endoom [4000]byte

// Demo. Not yet implemented.
type Demo struct{}

// DMXGUS. Not yet implemented.
type DMXGUS struct{}

// WAD eight-character string type. Null-terminated for short strings.
type String8 [8]byte

// String converts String8 to string
func (s String8) String() string {
	i := bytes.IndexByte(s[:], 0)
	if i == -1 {
		i = len(s)
	}
	return string(s[0:i])
}

// Special lump names
const SkyFlatName = "F_SKY1"

// /////////////////////////////////////
// NewWAD reads WAD metadata to memory. It returns a WAD object that
// can be used to read individual lumps.
// /////////////////////////////////////
func NewWAD(filename string) (*WAD, error) {
	logger.Println("Start reading WAD")

	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	wad := &WAD{file: file}

	// Read header
	var binHeader binHeader
	if err := binary.Read(file, binary.LittleEndian, &binHeader); err != nil {
		return nil, err
	}
	if string(binHeader.Magic[:]) != "IWAD" {
		return nil, fmt.Errorf("bad magic: %s", binHeader.Magic)
	}
	wad.header = &Header{int(binHeader.NumLumps), int(binHeader.InfoTableOfs)}

	// Read info tables
	if err := wad.readInfoTables(); err != nil {
		return nil, err
	}

	// Read PLAYPAL
	playpal, err := wad.readPlaypal()
	if err != nil {
		return nil, err
	}
	wad.Palettes = playpal
	wad.TransparentIndex = 255

	// Read COLORMAP
	colorMaps, err := wad.readColorMaps()
	if err != nil {
		return nil, err
	}
	wad.ColorMaps = colorMaps

	// Read ENDOOM
	endoom, err := wad.readEndoom()
	if err != nil {
		return nil, err
	}
	wad.Endoom = endoom

	// Read Demo lumps
	demos, err := wad.readDemos()
	if err != nil {
		return nil, err
	}
	wad.Demos = demos

	// Read DMXGUS
	dmxgus, err := wad.readDMXGUS()
	if err != nil {
		return nil, err
	}
	wad.Dmxgus = dmxgus

	// Read patch names
	wad.patchNames, err = wad.readPatchNames()
	if err != nil {
		return nil, err
	}

	// Read patchPics into Pictures map
	err = wad.readPatchPics()
	if err != nil {
		return nil, err
	}

	// Read map textures
	// Must be called after readPatchNames and readPatchLumps
	textures, texturesList, err := wad.readTextures()
	if err != nil {
		return nil, err
	}
	wad.Textures = textures
	wad.TexturesList = texturesList

	// Read flat lumps
	flats, flatsList, err := wad.readFlats()
	if err != nil {
		return nil, err
	}
	wad.Flats = flats
	wad.FlatsList = flatsList

	// Read sprite lumps
	sprites, err := wad.readSprites()
	if err != nil {
		return nil, err
	}
	wad.Sprites = sprites

	// Build sprite frames
	// spriteFrames, err := wad.buildSpriteFrames()
	// if err != nil {
	// 	return nil, err
	// }
	// wad.SpriteFrames = spriteFrames

	// Read sound lumps
	sounds, err := wad.readSounds()
	if err != nil {
		return nil, err
	}
	wad.Sounds = sounds

	// Read music lumps
	scores, err := wad.readMusic()
	if err != nil {
		return nil, err
	}
	wad.Scores = scores

	return wad, nil
}

func (w *WAD) readInfoTables() error {
	if err := w.seek(int64(w.header.InfoTableOfs)); err != nil {
		return err
	}
	lumpNums := map[string]int{}
	levels := map[string]int{}
	lumpInfos := make([]LumpInfo, w.header.NumLumps)
	for i := 0; i < w.header.NumLumps; i++ {
		var binInfo binLumpInfo
		if err := binary.Read(w.file, binary.LittleEndian, &binInfo); err != nil {
			return err
		}
		lumpInfo := LumpInfo{binInfo.Name.String(), int(binInfo.Filepos), int(binInfo.Size)}
		if lumpInfo.Name == "THINGS" {
			lumpNum := i - 1
			info := lumpInfos[lumpNum]
			levels[info.Name] = lumpNum
		}
		lumpNums[lumpInfo.Name] = i
		lumpInfos[i] = lumpInfo
	}
	w.levels = levels
	w.lumpNums = lumpNums
	w.lumpInfos = lumpInfos
	return nil
}

// readPlaypal
func (w *WAD) readPlaypal() (*Palettes, error) {
	logger.Println("Loading PLAYPAL ...")
	if err := w.seekLumpName("PLAYPAL"); err != nil {
		return nil, err
	}
	playpal := Palettes{}
	if err := binary.Read(w.file, binary.LittleEndian, &playpal); err != nil {
		return nil, err
	}
	return &playpal, nil
}

// readColorMaps
func (w *WAD) readColorMaps() (*ColorMaps, error) {
	logger.Println("Loading COLORMAP ...")
	if err := w.seekLumpName("COLORMAP"); err != nil {
		return nil, err
	}
	colormaps := ColorMaps{}
	if err := binary.Read(w.file, binary.LittleEndian, &colormaps); err != nil {
		return nil, err
	}
	return &colormaps, nil
}

// readEndoom reads the ENDOOM lump
func (w *WAD) readEndoom() (*Endoom, error) {
	logger.Println("Loading ENDOOM ...")
	if err := w.seekLumpName("ENDOOM"); err != nil {
		return nil, err
	}
	endoom := Endoom{}
	if err := binary.Read(w.file, binary.LittleEndian, &endoom); err != nil {
		return nil, err
	}
	return &endoom, nil
}

// readDemos reads all the DEMO lumps. Not yet implemented.
func (w *WAD) readDemos() ([]Demo, error) {
	return nil, nil
}

// readDemos reads all the DMXGUS lump. Not yet implemented.
func (w *WAD) readDMXGUS() (*DMXGUS, error) {
	return nil, nil
}

// readPatchNames reads the PNAMES lump to populate a slice of patch names
func (w *WAD) readPatchNames() ([]string, error) {
	logger.Printf("Loading patch names ...\n")
	if err := w.seekLumpName("PNAMES"); err != nil {
		return nil, err
	}

	// Read PNAMES header
	var count uint32
	if err := binary.Read(w.file, binary.LittleEndian, &count); err != nil {
		return nil, err
	}

	// Read and translate PNAMES body
	pnames := make([]String8, count)
	patchNames := make([]string, count)
	if err := binary.Read(w.file, binary.LittleEndian, pnames); err != nil {
		return nil, err
	}
	for i, p := range pnames {
		patchNames[i] = strings.ToUpper(p.String()) // ToUpper required for "w94_1" patch
	}
	return patchNames, nil
}

func (w *WAD) readPatchPics() error {
	logger.Println("Loading patch pictures ...")
	for _, pname := range w.patchNames {
		_, err := w.GetPicture(pname) // Also caches picture
		if err != nil {
			logger.Printf("Err: %v", err)
			continue
		}
	}
	logger.Printf("Loaded %v patch pictures", len(w.Pictures))

	return nil
}

func (w *WAD) readTextures() (map[string]*Texture, []*Texture, error) {
	logger.Println("Loading textures ...")

	textures := make(map[string]*Texture)
	texturesList := make([]*Texture, 0)
	for i := 1; i < 10; i++ {

		name := fmt.Sprintf("TEXTURE%v", i)

		lumpNum, ok := w.lumpNums[name]
		if !ok {
			continue
		}
		lumpInfo := w.lumpInfos[lumpNum]
		if err := w.seekLumpName(name); err != nil {
			continue
		}
		logger.Printf("Loading %v ...", name)

		// Read header
		var count uint32
		if err := binary.Read(w.file, binary.LittleEndian, &count); err != nil {
			return nil, nil, err
		}
		offsets := make([]int32, count)

		// Read offsets
		if err := binary.Read(w.file, binary.LittleEndian, offsets); err != nil {
			return nil, nil, err
		}

		// For each offset...
		for _, offset := range offsets {
			if err := w.seek(int64(lumpInfo.Filepos) + int64(offset)); err != nil {
				return nil, nil, err
			}

			// Read header
			var binHeader binTextureHeader
			if err := binary.Read(w.file, binary.LittleEndian, &binHeader); err != nil {
				return nil, nil, err
			}

			// Create texture
			texture := &Texture{
				Name:     binHeader.TextureName.String(),
				IsMasked: binHeader.Masked != 0,
				Width:    int(binHeader.Width),
				Height:   int(binHeader.Height),
			}

			// Add patches to texture
			binPatches := make([]binPatch, binHeader.NumPatches)
			patches := make([]Patch, binHeader.NumPatches)
			if err := binary.Read(w.file, binary.LittleEndian, binPatches); err != nil {
				return nil, nil, err
			}
			for pi, p := range binPatches {
				patches[pi] = Patch{
					XOffset: int(p.XOffset),
					YOffset: int(p.YOffset),
					Picture: w.Pictures[w.patchNames[p.PatchNameIdx]],
				}
			}
			texture.Patches = patches

			// Expand out patches to create composite Picture
			picture := &Picture{
				Name:       texture.Name,
				Width:      texture.Width,
				Height:     texture.Height,
				LeftOffset: 0,
				TopOffset:  0,
				Columns:    make([]Column, int(texture.Width)),
			}
			for i := range picture.Columns {
				picture.Columns[i] = make([]byte, int(texture.Height))
			}
			for _, p := range texture.Patches {
				sourceYOffset := 0
				if p.YOffset < 0 {
					sourceYOffset = -p.YOffset
					p.YOffset = 0
				}
				for y, c := range p.Picture.Columns {
					if p.XOffset+y >= 0 && p.XOffset+y < len(picture.Columns) {
						copy(picture.Columns[p.XOffset+y][p.YOffset:], c[sourceYOffset:])
					}
				}
			}
			texture.Picture = picture

			texture.Index = len(texturesList)
			textures[texture.Name] = texture
			texturesList = append(texturesList, texture)
		}
	}
	logger.Printf("Loaded %v textures", len(textures))

	return textures, texturesList, nil
}

// readFlats
func (w *WAD) readFlats() (map[string]*Flat, []*Flat, error) {
	logger.Println("Loading flats ...")

	flats := make(map[string]*Flat)
	flatsList := make([]*Flat, 0)
	startLump, ok := w.lumpNums["F_START"]
	if !ok {
		return nil, nil, fmt.Errorf("F_START not found")
	}
	endLump, ok := w.lumpNums["F_END"]
	if !ok {
		return nil, nil, fmt.Errorf("F_END not found")
	}

	// For each flat lump
	for i := startLump; i < endLump; i++ {
		lumpInfo := w.lumpInfos[i]

		// Skip marker lumps
		if lumpInfo.Size == 0 {
			continue
		}

		// Allocate Flat
		var flat Flat
		flat.Data = make([]byte, FlatHeight*FlatWidth)

		// Read lump and add to slice
		if err := w.seek(int64(lumpInfo.Filepos)); err != nil {
			return nil, nil, err
		}
		if err := binary.Read(w.file, binary.LittleEndian, flat.Data); err != nil {
			return nil, nil, err
		}

		flat.Name = lumpInfo.Name
		flat.Index = len(flatsList)
		flats[lumpInfo.Name] = &flat
		flatsList = append(flatsList, &flat)
	}
	logger.Printf("Loaded %v flats", len(flats))
	return flats, flatsList, nil
}

// readSounds
func (w *WAD) readSounds() (map[string]*Sound, error) {
	logger.Printf("Loading DS sounds ...")
	sounds := make(map[string]*Sound)

	// Check all lumps for sounds
	for _, li := range w.lumpInfos {

		// Skip non-sound lumps
		if li.Name[:2] != "DS" {
			continue
		}

		// Read header
		if err := w.seek(int64(li.Filepos)); err != nil {
			return nil, err
		}
		var header binSoundHeader
		if err := binary.Read(w.file, binary.LittleEndian, &header); err != nil {
			return nil, err
		}
		if header.Format != 3 {
			logger.Printf("Skipping unexpected sound format")
			continue
		}

		// Read the samples
		numSamples := header.Bytes - 32 // Subtract 32 pad bytes
		samples := make([]byte, numSamples)
		if err := binary.Read(w.file, binary.LittleEndian, samples); err != nil {
			return nil, err
		}
		sounds[li.Name] = &Sound{
			SampleRate: uint(header.SampleRate),
			Samples:    samples,
		}
	}
	logger.Printf("Loaded %v sounds", len(sounds))
	return sounds, nil
}

// readSounds
func (w *WAD) readMusic() (map[string]*MusicScore, error) {
	logger.Printf("Loading music ...")
	scores := make(map[string]*MusicScore)

	// Check all lumps for music
	for _, li := range w.lumpInfos {

		// Skip non-sound lumps
		if li.Name[:2] != "D_" {
			continue
		}

		// Read header
		if err := w.seek(int64(li.Filepos)); err != nil {
			return nil, err
		}
		var header binMusicHeader
		if err := binary.Read(w.file, binary.LittleEndian, &header); err != nil {
			return nil, err
		}

		// Read the instruments
		samples := make(binMusicInstruments, header.InstrumentCount)
		if err := binary.Read(w.file, binary.LittleEndian, samples); err != nil {
			return nil, err
		}

		// Read sound events

		// 	scores[li.Name] = &Score{
		// 		SampleRate: uint(header.SampleRate),
		// 		Samples:    samples,
		// 	}
	}
	logger.Printf("Loaded %v scores", len(scores))
	return scores, nil
}

// readSprites
// A Sprite is a slice of SpriteFrames
// A SpriteFrame is eight Sprite Pictures, for each direction
// A Sprite Picture is just a Doom Picture
func (w *WAD) readSprites() (map[string]*Sprite, error) {
	logger.Println("Loading sprites ...")
	sprites := make(map[string]*Sprite)

	// Find start and end lumps
	startLump, ok := w.lumpNums["S_START"]
	if !ok {
		return nil, fmt.Errorf("S_START not found")
	}
	endLump, ok := w.lumpNums["S_END"]
	if !ok {
		return nil, fmt.Errorf("S_END not found")
	}

	// For each sprite picture lump
	for i := startLump; i < endLump; i++ {
		lumpInfo := w.lumpInfos[i]

		// Skip marker lumps
		if lumpInfo.Size == 0 {
			continue
		}

		// Read lump into Picture format
		picture, err := w.GetPicture(lumpInfo.Name)
		if err != nil {
			logger.Printf("Err: %v", err)
			continue
		}

		// Construct sprite name
		spriteName := lumpInfo.Name[:4]
		spriteframe := int(lumpInfo.Name[4] - 'A')
		sprite, ok := sprites[spriteName]
		if !ok {
			sprite = new(Sprite)
		}

		// Grow sprite slice to fit this slice frame
		for (len(*sprite) - 1) < spriteframe {
			*sprite = append(*sprite, SpriteFrame{})
		}
		sf := &(*sprite)[spriteframe]

		// If rotation zero, use this picture for all sprite directions
		rotation := lumpInfo.Name[5] - '1'
		if rotation == 0xff {
			for i := range 8 {
				sf[i].Picture = picture
			}
		} else {
			sf[rotation].Picture = picture
		}

		if len(lumpInfo.Name) >= 8 {
			if lumpInfo.Name[4] != lumpInfo.Name[6] {
				logger.Println("ERR: Frames mismatch:", lumpInfo.Name)
				continue
			}
			rotation := lumpInfo.Name[7] - '1'
			if rotation == 0xff {
				logger.Println("ERR: Flipped all rotation:", lumpInfo.Name)
				continue
			}
			sf[rotation].Picture = picture
			sf[rotation].IsFlipped = true
		}
		sprites[spriteName] = sprite

	}
	logger.Printf("Loaded %v sprites", len(sprites))
	logger.Printf("(Loaded %v pictures)", len(w.Pictures))
	return sprites, nil
}

// LevelNames returns a slice of level names found in the WAD archive.
func (w *WAD) LevelNames() []string {
	result := make([]string, 0, len(w.levels))
	for name := range w.levels {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// ReadLevel reads level data from WAD archive and returns a Level struct.
func (w *WAD) ReadLevel(name string, sectorUser any) (*Level, error) {
	logger.Printf("Reading Level %v ...", name)

	level := Level{}
	levelIdx := w.levels[name]
	for i := levelIdx + 1; i < levelIdx+11; i++ {
		lumpInfo := w.lumpInfos[i]
		if err := w.seek(int64(lumpInfo.Filepos)); err != nil {
			return nil, err
		}
		name := lumpInfo.Name
		switch name {
		case "THINGS":
			things, err := w.readThings(&lumpInfo)
			if err != nil {
				return nil, err
			}
			level.Things = things
		case "SIDEDEFS":
			sides, err := w.readSides(&lumpInfo)
			if err != nil {
				return nil, err
			}
			level.Sides = sides
		case "LINEDEFS":
			lines, err := w.readLines(&lumpInfo)
			if err != nil {
				return nil, err
			}
			level.Lines = lines
		case "VERTEXES":
			vertexes, err := w.readVertexes(&lumpInfo)
			if err != nil {
				return nil, err
			}
			level.Vertexes = vertexes
		case "SEGS":
			segments, err := w.readLineSegments(&lumpInfo)
			if err != nil {
				return nil, err
			}
			level.LineSegments = segments
		case "SSECTORS":
			subsectors, err := w.readSubSectors(&lumpInfo)
			if err != nil {
				return nil, err
			}
			level.SubSectors = subsectors
		case "NODES":
			nodes, err := w.readNodes(&lumpInfo)
			if err != nil {
				return nil, err
			}
			level.Nodes = nodes
		case "SECTORS":
			sectors, err := w.readSectors(&lumpInfo, sectorUser)
			if err != nil {
				return nil, err
			}
			level.Sectors = sectors
		case "REJECT":
			reject, err := w.readReject(&lumpInfo)
			if err != nil {
				return nil, err
			}
			level.Reject = *reject
		case "BLOCKMAP":
			blockMap, err := w.readBlockmap(&lumpInfo)
			if err != nil {
				return nil, err
			}
			level.BlockMap = *blockMap
		default:
			logger.Printf("Unhandled lump %s\n", name)
		}
	}

	// Set references
	w.setReferences(&level)

	return &level, nil
}

// setReferences adds pointers to all level assets
func (w *WAD) setReferences(l *Level) error {
	logger.Println("Setting references ...")

	// Sides
	for i := range l.Sides {
		l.Sides[i].Sector = &l.Sectors[l.Sides[i].SectorNum]
	}

	// Lines - dependent on Sides
	for i := range l.Lines {
		li := &l.Lines[i] // Point to element
		li.V1 = l.Vertexes[li.V1Num]
		li.V2 = l.Vertexes[li.V2Num]
		li.DX = li.V2.X - li.V1.X
		li.DY = li.V2.Y - li.V1.Y
		if li.SideRNum >= 0 { // -1 means no Side
			li.SideR = &l.Sides[li.SideRNum]
			li.FrontSector = li.SideR.Sector
		}
		if li.SideLNum >= 0 { // -1 means no Side
			li.SideL = &l.Sides[li.SideLNum]
			li.BackSector = li.SideL.Sector
		}

		// Point to tagged sectors
		for j := range l.Sectors {
			if l.Sectors[j].TagNum == li.SectorTagNum {
				li.TaggedSectors = append(li.TaggedSectors, &l.Sectors[j])
			}
		}

		// Set slope type
		if li.DX == 0 {
			li.SlopeType = SlopeTypeVertical
		} else if li.DY == 0 {
			li.SlopeType = SlopeTypeHorizontal
		} else if (li.DY / li.DX) > 0 {
			li.SlopeType = SlopeTypePositive
		} else {
			li.SlopeType = SlopeTypeNegative
		}

		// Set bounding box
		li.BoundingBox.Left = min(li.V1.X, li.V2.X)
		li.BoundingBox.Right = max(li.V1.X, li.V2.X)
		li.BoundingBox.Bottom = min(li.V1.Y, li.V2.Y)
		li.BoundingBox.Left = max(li.V1.Y, li.V2.Y)

	}

	// Line Segments
	for i := range l.LineSegments {
		s := &l.LineSegments[i] // Point to element
		s.V1 = l.Vertexes[s.V1Num]
		s.V2 = l.Vertexes[s.V2Num]
		s.Line = &l.Lines[s.LineNum]
		if s.IsSideL {
			s.Side = &l.Sides[s.Line.SideLNum]
		} else {
			s.Side = &l.Sides[s.Line.SideRNum]
		}
		s.FrontSector = s.Side.Sector
		if s.Line.TwoSided {
			if s.IsSideL {
				s.BackSector = l.Sides[s.Line.SideRNum].Sector
			} else {
				s.BackSector = l.Sides[s.Line.SideLNum].Sector
			}
		}
	}

	// SubSectors
	for i := range l.SubSectors {
		s := &l.SubSectors[i] // Point to element
		for j := s.StartLineSegment; j < (s.StartLineSegment + s.numLineSegments); j++ {
			s.LineSegments = append(s.LineSegments, l.LineSegments[j])
		}
		s.Sector = s.LineSegments[0].Side.Sector
	}

	// Nodes
	l.RootNode = &l.Nodes[len(l.Nodes)-1]
	for i := range l.Nodes {
		n := &l.Nodes[i] // Point to element

		if n.ChildNumR < 0 {
			n.ChildR = &l.SubSectors[n.ChildNumR&math.MaxInt16]
		} else {
			n.ChildR = &l.Nodes[n.ChildNumR]
		}

		if n.ChildNumL < 0 {
			n.ChildL = &l.SubSectors[n.ChildNumL&math.MaxInt16]
		} else {
			n.ChildL = &l.Nodes[n.ChildNumL]
		}
	}

	// Sectors
	for i := range l.Sectors {
		s := &l.Sectors[i] // Point to element
		bbox := newBBox()
		for j := range l.Lines {
			l := &l.Lines[j] // Point to element
			if l.FrontSector == s || l.BackSector == s {
				s.Lines = append(s.Lines, l)
				bbox.add(l.V1)
				bbox.add(l.V2)
			}
		}

		// set the degenmobj_t to the middle of the bounding box
		s.SoundOrigin.X = (bbox.Right + bbox.Left) / 2
		s.SoundOrigin.Y = (bbox.Top + bbox.Bottom) / 2

		// adjust bounding box to map blocks
		block := int(bbox.Top - l.BlockMap.OriginY + MaxRadius)
		s.BlockBox.Top = min(block, l.BlockMap.NumRows-1)

		block = int(bbox.Bottom - l.BlockMap.OriginY - MaxRadius)
		s.BlockBox.Bottom = max(block, 0)

		block = int(bbox.Right - l.BlockMap.OriginX + MaxRadius)
		s.BlockBox.Right = min(block, l.BlockMap.NumColumns-1)

		block = int(bbox.Left - l.BlockMap.OriginX - MaxRadius)
		s.BlockBox.Right = max(block, 0)

	}

	// Block map
	for i := range l.BlockMap.Blocks {
		b := &l.BlockMap.Blocks[i] // Point to element
		for j := range b.LineNums {
			b.Lines = append(b.Lines, &l.Lines[b.LineNums[j]])
		}
	}

	// Soundtraversed int         // 0 = untraversed, 1,2 = sndlines -1
	// Soundtarget    *Mobj       // thing that made a sound (or null)
	// Blockbox       BBox        // mapblock bounding box for height changes
	// Soundorg       Degenmobj_t // origin for any sounds played by the sector
	// Validcount     int         // if == validcount, already checked
	// Thinglist      *Mobj       // Root of mobjs in sector linked list	// TODO - or should it be slice?
	// Specialdata    *Thinker    // thinker_t for reversable actions

	// Reject
	// TODO

	// Block Map
	// TODO

	return nil
}

// MaxRadius is for precalculated sector block boxes
// the spider demon is larger, but don't have any moving sectors nearby
const MaxRadius = 32

func newBBox() *BoundBox {
	return &BoundBox{
		Left:   math.MaxInt,
		Right:  math.MinInt,
		Bottom: math.MaxInt,
		Top:    math.MinInt,
	}
}

func (b *BoundBox) add(v Vertex) {
	b.Left = min(b.Left, v.X)
	b.Right = max(b.Right, v.X)
	b.Bottom = min(b.Bottom, v.Y)
	b.Top = min(b.Top, v.Y)
}

// func bBoxFromBin(b binBBox) BBox {
// 	return BBox{
// 		Top:    float64(b.Top),
// 		Bottom: float64(b.Bottom),
// 		Left:   float64(b.Left),
// 		Right:  float64(b.Right),
// 	}
// }

// func newSubSector(wadSubSector binSubSector, wadLevel *level, linedefs []*Line) SubSector {
// 	subSector := SubSector{Segments: make([]Segment, 0, wadSubSector.Numsegs)}
// 	for i := wadSubSector.StartSeg; i < wadSubSector.StartSeg+wadSubSector.Numsegs; i++ {
// 		seg := wadLevel.segs[i]
// 		vs := wadLevel.vertexes[seg.VertexStart]
// 		ve := wadLevel.vertexes[seg.VertexEnd]
// 		line := linedefs[seg.LineNum]
// 		side := line.SidedefLeft
// 		if seg.Direction == 1 {
// 			side = line.SideR
// 		}
// 		subSector.Segments = append(subSector.Segments, Segment{
// 			VertexStart: newVertex(vs),
// 			VertexEnd:   newVertex(ve),
// 			Angle:       float64(seg.Angle),
// 			Line:        line,
// 			Side:        side,
// 			Offset:      float64(seg.Offset),
// 		})
// 	}
// 	subSector.Sector = subSector.Segments[0].Side.Sector
// 	return subSector

// }

func (w *WAD) readThings(lumpInfo *LumpInfo) ([]Thing, error) {
	logger.Println("Reading Things ...")

	// Read things lump
	count := lumpInfo.Size / int(unsafe.Sizeof(binThing{}))
	binThings := make([]binThing, count)
	things := make([]Thing, count)
	if err := binary.Read(w.file, binary.LittleEndian, binThings); err != nil {
		return nil, err
	}

	// Translate to canonical
	for i, t := range binThings {
		things[i] = Thing{
			X:               int(t.X),
			Y:               int(t.Y),
			Angle:           degreesToRadians(t.Angle),
			Type:            int(t.Type),
			Skill1and2:      t.Options&1 != 0,
			Skill3:          t.Options&2 != 0,
			Skill4and5:      t.Options&4 != 0,
			Ambush:          t.Options&8 != 0,
			MultiplayerOnly: t.Options&0x10 != 0,
		}
	}
	logger.Printf("Read %v things", len(things))
	return things, nil
}

func (w *WAD) readLines(lumpInfo *LumpInfo) ([]Line, error) {
	logger.Println("Reading Lines ...")

	// Read lump
	count := lumpInfo.Size / int(unsafe.Sizeof(binLine{}))
	binLine := make([]binLine, count)
	lines := make([]Line, count)
	if err := binary.Read(w.file, binary.LittleEndian, binLine); err != nil {
		return nil, err
	}

	// Translate to canonical
	for i, line := range binLine {
		lines[i] = Line{
			V1Num:                  int(line.VertexStart),
			V2Num:                  int(line.VertexEnd),
			BlockPlayerAndMonsters: binLine[i].Flags&1 != 0,
			BlockMonsters:          line.Flags&2 != 0,
			TwoSided:               line.Flags&4 != 0,
			UpperTextureUnpegged:   line.Flags&8 != 0,
			LowerTextureUnpegged:   line.Flags&0x10 != 0,
			Secret:                 line.Flags&0x20 != 0,
			BlocksSound:            line.Flags&0x40 != 0,
			NeverMap:               line.Flags&0x80 != 0,
			AlwaysMap:              line.Flags&0x100 != 0,
			Type:                   LineType(line.Type),
			SectorTagNum:           int(line.SectorTag),
			SideRNum:               int(line.SideR),
			SideLNum:               int(line.SideL),
		}
	}

	logger.Printf("Read %v lines", len(lines))

	return lines, nil
}

func (w *WAD) readSides(lumpInfo *LumpInfo) ([]Side, error) {
	logger.Println("Reading Sides ...")

	// Read lump
	count := lumpInfo.Size / int(unsafe.Sizeof(binSide{}))
	binSides := make([]binSide, count)
	sides := make([]Side, count)
	if err := binary.Read(w.file, binary.LittleEndian, binSides); err != nil {
		return nil, err
	}

	// Translate to canonical
	for i, s := range binSides {
		sides[i] = Side{
			XOffset:           float64(s.XOffset),
			YOffset:           float64(s.YOffset),
			UpperTextureName:  s.UpperTexture.String(),
			MiddleTextureName: s.MiddleTexture.String(),
			LowerTextureName:  s.LowerTexture.String(),
			SectorNum:         int(s.SectorNum),
		}
		sides[i].UpperTexture = w.Textures[sides[i].UpperTextureName]
		sides[i].MiddleTexture = w.Textures[sides[i].MiddleTextureName]
		sides[i].LowerTexture = w.Textures[sides[i].LowerTextureName]
	}

	logger.Printf("Read %v sides", len(sides))
	return sides, nil
}

func (w *WAD) readVertexes(lumpInfo *LumpInfo) ([]Vertex, error) {
	logger.Println("Reading Vertexes ...")

	// Read lump
	count := lumpInfo.Size / int(unsafe.Sizeof(binVertex{}))
	binVertexes := make([]binVertex, count)
	vertexes := make([]Vertex, count)
	if err := binary.Read(w.file, binary.LittleEndian, binVertexes); err != nil {
		return nil, err
	}

	// Translate to canonical
	for i, v := range binVertexes {
		vertexes[i] = Vertex{X: float64(v.X), Y: float64(v.Y)}
	}
	logger.Printf("Read %v vertexes", len(vertexes))

	return vertexes, nil
}

func (w *WAD) readLineSegments(lumpInfo *LumpInfo) ([]LineSegment, error) {
	logger.Println("Reading Line Segments ...")

	// Read lump
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(binLineSegment{}))
	binSegments := make([]binLineSegment, count)
	segments := make([]LineSegment, count)
	if err := binary.Read(w.file, binary.LittleEndian, binSegments); err != nil {
		return nil, err
	}

	// Translate to canonical
	for i, s := range binSegments {
		segments[i] = LineSegment{
			V1Num:   int(s.V1),
			V2Num:   int(s.V2),
			Angle:   bamToRadians(s.Angle),
			LineNum: int(s.LineNum),
			IsSideL: s.Direction == 1,
			Offset:  float64(s.Offset),
		}
	}
	logger.Printf("Read %v line segments", len(segments))

	return segments, nil
}

func (w *WAD) readSubSectors(lumpInfo *LumpInfo) ([]SubSector, error) {
	logger.Println("Reading Sub Sectors ...")

	// Read lump
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(binSubSector{}))
	binSubSectors := make([]binSubSector, count)
	subSectors := make([]SubSector, count)
	if err := binary.Read(w.file, binary.LittleEndian, binSubSectors); err != nil {
		return nil, err
	}

	// Translate to canonical
	for i, s := range binSubSectors {
		subSectors[i] = SubSector{
			numLineSegments:  int(s.NumSegments),
			StartLineSegment: int(s.StartLineSegment),
		}
	}
	logger.Printf("Read %v sub sectors", len(subSectors))

	return subSectors, nil
}

func (w *WAD) readNodes(lumpInfo *LumpInfo) ([]Node, error) {
	logger.Println("Reading Nodes ...")

	// Read lump
	count := lumpInfo.Size / int(unsafe.Sizeof(binNode{}))
	binNodes := make([]binNode, count)
	nodes := make([]Node, count)
	if err := binary.Read(w.file, binary.LittleEndian, binNodes); err != nil {
		return nil, err
	}

	// Translate to canonical
	for i, n := range binNodes {
		nodes[i] = Node{
			X:  float64(n.X),
			Y:  float64(n.Y),
			DX: float64(n.DX),
			DY: float64(n.DY),
			BBoxR: BoundBox{
				float64(n.BBoxR.Top),
				float64(n.BBoxR.Bottom),
				float64(n.BBoxR.Left),
				float64(n.BBoxR.Right),
			},
			BBoxL: BoundBox{
				float64(n.BBoxL.Top),
				float64(n.BBoxL.Bottom),
				float64(n.BBoxL.Left),
				float64(n.BBoxL.Right),
			},
			ChildNumR: int(n.ChildNumR),
			ChildNumL: int(n.ChildNumL),
		}
	}
	logger.Printf("Read %v nodes", len(nodes))

	return nodes, nil
}

func (w *WAD) readSectors(lumpInfo *LumpInfo, sectorUser any) ([]Sector, error) {
	logger.Println("Reading Sectors ...")

	// Read lump
	count := lumpInfo.Size / int(unsafe.Sizeof(binSector{}))
	binSectors := make([]binSector, count)
	sectors := make([]Sector, count)
	if err := binary.Read(w.file, binary.LittleEndian, binSectors); err != nil {
		return nil, err
	}

	// Translate to canonical
	for i, s := range binSectors {
		newUser, err := cloneSectorUserData(sectorUser)
		if err != nil {
			return nil, errors.New("cannot clone passed sectorUserData")
		}
		sectors[i] = Sector{
			Index:              i,
			FloorHeight:        float64(s.FloorHeight),
			CeilingHeight:      float64(s.CeilingHeight),
			FloorTextureName:   s.FloorTexture.String(),
			CeilingTextureName: s.CeilingTexture.String(),
			LightLevel:         int(s.LightLevel),
			Type:               SectorType(s.Type),
			TagNum:             int(s.TagNum),
			User:               newUser,
		}
		sectors[i].FloorTexture = w.Flats[sectors[i].FloorTextureName]
		sectors[i].CeilingTexture = w.Flats[sectors[i].CeilingTextureName]
	}
	logger.Printf("Read %v Sectors", len(sectors))

	return sectors, nil
}

func (w *WAD) readReject(lumpInfo *LumpInfo) (*Reject, error) {
	logger.Println("Reading Reject ...")

	// Read lump
	lump, err := w.readLump(lumpInfo)
	if err != nil {
		return nil, err
	}

	numSectors := int(math.Sqrt(float64(8 * lumpInfo.Size)))
	reject := make(Reject, numSectors)
	for sector1 := range numSectors {
		reject[sector1] = make([]bool, numSectors)
		for sector2 := range numSectors {
			cell := sector1*numSectors + sector2
			i, j := cell/8, cell%8
			if (lump[i] << j) > 0 {
				reject[sector1][sector2] = true
			}
		}
	}
	logger.Printf("Read Reject table: %v sectors", len(reject))

	return &reject, nil

}

func (w *WAD) readBlockmap(lumpInfo *LumpInfo) (*BlockMap, error) {
	logger.Println("Reading Block Map ...")

	// Read lump
	lump, err := w.readLump(lumpInfo)
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer(lump)

	// Read header
	var header binBlockMapHeader
	if err := binary.Read(buffer, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	// Read offsets - count of int16s to skip
	offsets := make([]binBlockListOffset, header.Columns*header.Rows)
	if err := binary.Read(buffer, binary.LittleEndian, offsets); err != nil {
		return nil, err
	}

	// Populate block map header
	blockMap := BlockMap{
		OriginX:    float64(header.OriginX),
		OriginY:    float64(header.OriginY),
		NumColumns: int(header.Columns),
		NumRows:    int(header.Rows),
	}

	// Populate block lists
	for _, o := range offsets {
		reader := bytes.NewBuffer(lump[2*o:])
		var binlineNum binBlockLineNum
		lineNums := make([]int, 0)
		for binlineNum != 0xffff {
			if err := binary.Read(reader, binary.LittleEndian, &binlineNum); err != nil {
				return nil, err
			}
			if binlineNum != 0 && binlineNum != 0xffff {
				lineNums = append(lineNums, int(binlineNum))
			}
		}
		// if len(blockList) > 0 {
		blockMap.Blocks = append(blockMap.Blocks, Block{LineNums: lineNums})
		// }
	}
	logger.Printf("Read %v blocks", len(blockMap.Blocks))

	return &blockMap, nil
}

// seekLumpName
func (w *WAD) seekLumpName(name string) error {
	pnamesLump, ok := w.lumpNums[name]
	if !ok {
		return errors.New("lump not found")
	}
	lumpInfo := w.lumpInfos[pnamesLump]
	return w.seek(int64(lumpInfo.Filepos))
}

// seek
func (w *WAD) seek(offset int64) error {
	off, err := w.file.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	if off != offset {
		return fmt.Errorf("seek failed")
	}
	return nil
}

// Read entire lump
func (w *WAD) readLump(lumpInfo *LumpInfo) ([]byte, error) {
	if err := w.seek(int64(lumpInfo.Filepos)); err != nil {
		return nil, err
	}
	lump := make([]byte, lumpInfo.Size)
	n, err := w.file.Read(lump)
	if err != nil {
		return nil, err
	}
	if n != int(lumpInfo.Size) {
		return nil, fmt.Errorf("truncated lump")
	}
	return lump, nil
}

// degreesToRadians
func degreesToRadians[T constraints.Integer | constraints.Float](n T) float64 {
	return float64(n) * (math.Pi / 180)
}

const halfScale = 1 << 15

func bamToRadians[T constraints.Signed](n T) float64 {
	return ((float64(n) + halfScale) * math.Pi) / halfScale
}

// CloneStruct clones a struct referenced by an any interface
func cloneSectorUserData(src any) (any, error) {
	// Get the value of the source
	srcVal := reflect.ValueOf(src)

	// Ensure the source is a struct
	if srcVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("source is not a struct")
	}

	// Create a new instance of the same type
	cloneVal := reflect.New(srcVal.Type()).Elem()

	// Copy the fields from the source struct to the new instance
	cloneVal.Set(srcVal)

	// Return the new instance as an any interface
	return cloneVal.Interface(), nil
}
