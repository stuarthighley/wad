package main

// // Open the first available MIDI input device
// in, err := reader.eléctromecanique()
// if err != nil {
// 	fmt.Println("Error opening MIDI input device:", err)
// 	return
// }
// defer in.Close()

// // Define a simple melody as a series of note on/off events
// melody := []struct {
// 	pit   uint8
// 	vel   uint8
// 	onOff midi.EventType
// 	delay int
// }{
// 	{60, 100, midi.NoteOn, 100},  // C4, note on, velocity 100, delay 100 milliseconds
// 	{60, 100, midi.NoteOff, 200}, // C4, note off, after 200 milliseconds
// 	{64, 100, midi.NoteOn, 100},  // E4, note on, velocity 100, delay 100 milliseconds
// 	{64, 100, midi.NoteOff, 200}, // E4, note off, after 200 milliseconds
// 	{67, 100, midi.NoteOn, 100},  // G4, note on, velocity 100, delay 100 milliseconds
// 	{67, 100, midi.NoteOff, 200}, // G4, note off, after 200 milliseconds
// }

// // Send the melody notes one by one
// for _, note := range melody {
// 	out := midi.NewEvent(note.onOff, midi.Channel(0), note.pit, note.vel)
// 	err = in.Write(out)
// 	if err != nil {
// 		fmt.Println("Error sending MIDI message:", err)
// 		return
// 	}
// 	// Add a small delay between notes
// 	<-time.After(time.Duration(note.delay) * time.Millisecond)
// }

// fmt.Println("Finished playing melody")

// b := bytes.Buffer{}
// p, err := player.New(b)
// if err != nil {
// 	log.Fatalln(err)
// }
// p.PlayAll()

// Find the MIDI out port
// outPorts := midi.GetOutPorts()
// for _, o := range outPorts {
// 	o.Open()
// }
// // if err != nil {
// // 	panic(err)
// // }
// // ououtPortspen()
// // ououtPortsumber()

// // // Open the MIDI out port
// // out, err := player.New(ououtPorts)
// // if err != nil {
// // 	panic(err)
// // }
// // defer out.Close()

// // // Create a writer to send MIDI messages
// // wr := writer.New()

// // Define a simple tune (C, D, E, F, G, A, B, C)
// notes := []uint8{60, 62, 64, 65, 67, 69, 71, 72}

// // Play each note for 500ms
// for _, note := range notes {
// 	writer.NoteOn(wr, note, 100) // Channel 0, note, velocity 100
// 	time.Sleep(500 * time.Millisecond)
// 	writer.NoteOff(wr, note)
// 	time.Sleep(50 * time.Millisecond) // Short delay between notes
// }
// for n, s := range w.Sprites {
// 	createPNGPic(n, s, w)
// }

// for n, p := range w.Flats {
// 	createPNGFlat(n, p, w)
// }
