package wad

type binLine struct {
	VertexStart, VertexEnd int16
	Flags                  int16
	Type                   int16
	SectorTag              int16
	SideR, SideL           int16
}

type Line struct {
	V1Num                  int
	V2Num                  int
	BlockPlayerAndMonsters bool
	BlockMonsters          bool
	TwoSided               bool
	UpperTextureUnpegged   bool
	LowerTextureUnpegged   bool
	Secret                 bool
	BlocksSound            bool
	NeverMap               bool
	AlwaysMap              bool
	Type                   LineType
	SectorTagNum           int
	SideRNum, SideLNum     int

	// References
	V1, V2                  Vertex
	DX, DY                  float64 // Precalculated VertexEnd-VertexStart for side checking
	TaggedSectors           []*Sector
	SideR, SideL            *Side     // One of these can be null if one-sided. Not sure
	BoundingBox             BoundBox  // For the extent of the LineDef
	SlopeType               SlopeType // To aid move clipping
	FrontSector, BackSector *Sector   // Redundant? Can be retrieved from Sides
	ValidCount              int       // if == validcount, already checked
	// SpecialData             Thinker   // Thinker for reversable actions	// Unused on Line? TODO
}

type SlopeType int

const (
	SlopeTypeHorizontal SlopeType = iota
	SlopeTypeVertical
	SlopeTypePositive
	SlopeTypeNegative
)

// Type	Description
// 1	DR Door
// 2	W1 Door Stay Open
// 3	W1 Door Close
// 4	W1 Door
// 5	W1 Floor To Lowest Adjacent Ceiling
// 6	W1 Start Crusher, Fast Damage
// 7	S1 Build Stairs 8 Up
// 8	W1 Build Stairs 8 Up
// 9	S1 Floor Donut
// 10	W1 Lift Also Monsters
// 11	S1 Exit (Normal)
// 12	W1 Light To Highest Adjacent Level
// 13	W1 Light To 255
// 14	S1 Floor Up 32 Change Texture
// 15	S1 Floor Up 24 Change Texture
// 16	W1 Door Close and Open
// 17	W1 Light Blink 1.0 Sec
// 18	S1 Floor To Higher Adjacent Floor
// 19	W1 Floor To Highest Adjacent Floor
// 20	S1 Floor To Higher Floor Change Texture
// 21	S1 Lift
// 22	W1 Floor To Higher Floor Change Texture
// 23	S1 Floor To Lowest Adjacent Floor
// 24	G1 Floor To Lowest Adjacent Ceiling
// 25	W1 Start Crusher, Slow Damage
// 26	DR Door Blue Key
// 27	DR Door Yellow Key
// 28	DR Door Red Key
// 29	S1 Door
// 30	W1 Floor Up Shortest Lower Texture
// 31	D1 Door Stay Open
// 32	D1 Door Blue Key
// 33	D1 Door Red Key
// 34	D1 Door Yellow Key
// 35	W1 Light To 35
// 36	W1 Floor To 8 Above Highest Adjacent Floor Fast
// 37	W1 Floor To Lowest Adjacent Floor Change Texture and Type
// 38	W1 Floor To Lowest Adjacent Floor
// 39	W1 Teleport
// 40	W1 Ceiling To Highest Ceiling
// 41	S1 Ceiling To Floor
// 42	SR Door Close
// 43	SR Ceiling To Floor
// 44	W1 Ceiling To 8 Above Floor
// 45	SR Floor To Highest Adjacent Floor
// 46	GR Door Also Monsters
// 47	G1 Floor To Higher Floor Change Texture
// 48	Scrolling Wall Left
// 49	S1 Start Crusher, Slow Damage
// 50	S1 Door Close
// 51	S1 Exit (Secret)
// 52	W1 Exit (Normal)
// 53	W1 Start Moving Floor
// 54	W1 Stop Moving Floor
// 55	S1 Floor To 8 Below Lowest Adjacent Ceiling and Crush
// 56	W1 Floor To 8 Below Lowest Adjacent Ceiling and Crush
// 57	W1 Stop Crusher
// 58	W1 Floor Up 24
// 59	W1 Floor Up 24 Change Texture and Type
// 60	SR Floor To Lowest Adjacent Floor
// 61	SR Door Stay Open
// 62	SR Lift
// 63	SR Door
// 64	SR Floor To Lowest Adjacent Ceiling
// 65	SR Floor To 8 Below Lowest Adjacent Ceiling and Crush
// 66	SR Floor Up 24 Change Texture
// 67	SR Floor Up 32 Change Texture
// 68	SR Floor To Higher Floor Change Texture
// 69	SR Floor To Higher Adjacent Floor
// 70	SR Floor To 8 Above Higher Adjacent Floor Fast
// 71	S1 Floor To 8 Above Higher Adjacent Floor Fast
// 72	WR Ceiling To 8 Above Floor
// 73	WR Start Crusher, Slow Damage
// 74	WR Stop Crusher
// 75	WR Door Close
// 76	WR Door Close and Open
// 77	WR Start Crusher, Fast Damage
// 79	WR Light To 35
// 80	WR Light To Highest Adjacent Level
// 81	WR Light To 255
// 82	WR Floor To Lowest Adjacent Floor
// 83	WR Floor To Highest Adjacent Floor
// 84	WR Floor To Lowest Adjacent Floor Change Texture and Type
// 86	WR Door Stay Open
// 87	WR Start Moving Floor
// 88	WR Lift Also Monsters
// 89	WR Stop Moving Floor
// 90	WR Door
// 91	WR Floor To Lowest Adjacent Ceiling
// 92	WR Floor Up 24
// 93	WR Floor Up 24 Change Texture and Type
// 94	WR Floor To 8 Below Lowest Adjacent Ceiling and Crush
// 95	WR Floor To Higher Floor Change Texture
// 96	WR Floor Up Shortest Lower Texture
// 97	WR Teleport
// 98	WR Floor To 8 Above Highest Adjacent Floor Fast
// 99	SR Door Blue Key Fast
// 100	W1 Build Stairs 16 and Crush
// 101	S1 Floor To Lowest Adjacent Ceiling
// 102	S1 Floor To Highest Adjacent Floor
// 103	S1 Door Stay Open
// 104	W1 Light To Lowest Adjacent Level
// 105	WR Door Fast
// 106	WR Door Stay Open Fast
// 107	WR Door Close Fast
// 108	W1 Door Fast
// 109	W1 Door Stay Open Fast
// 110	W1 Door Close Fast
// 111	S1 Door Fast
// 112	S1 Door Stay Open Fast
// 113	S1 Door Close Fast
// 114	SR Door Fast
// 115	SR Door Stay Open Fast
// 116	SR Door Close Fast
// 117	DR Door Fast
// 118	D1 Door Fast
// 119	W1 Floor To Higher Adjacent Floor
// 120	WR Lift Fast
// 121	W1 Lift Fast
// 122	S1 Lift Fast
// 123	SR Lift Fast
// 124	W1 Exit (Secret)
// 125	W1 Teleport Monsters Only
// 126	WR Teleport Monsters Only
// 127	S1 Build Stairs 16 + Crush
// 128	WR Floor To Higher Adjacent Floor
// 129	WR Floor To Higher Floor Fast
// 130	W1 Floor To Higher Floor Fast
// 131	S1 Floor To Higher Floor Fast
// 132	SR Floor To Higher Floor Fast
// 133	S1 Door Blue Key Fast
// 134	SR Door Red Key Fast
// 135	S1 Door Red Key Fast
// 136	SR Door Yellow Key Fast
// 137	S1 Door Yellow Key Fast
// 138	SR Light To 255
// 139	SR Light To 35
// 140	S1 Floor Up 512
// 141	W1 Start Crusher, Silent

type LineType int

const (
// TODO
)
