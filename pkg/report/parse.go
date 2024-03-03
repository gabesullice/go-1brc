package report

import "fmt"

// temperature stores a decimal number between 0.0 and 99.9 inclusive as 3 uint8 values, representing the tens, ones,
// and tenths places in the 0, 1, and 2 index, respectively.
type temperature [3]uint8

type reading struct {
	station []byte
	subzero bool
	temp    temperature
}

const noNewline = -1

const lenMinReading = len("x;0.0\n")

func parseLeftRightBytes(d []byte, readings chan<- *reading) {
	splitAt := len(d) / 2
	initialNLRight, _ := parseBytes(d[splitAt:], readings)
	initialNLLeft, terminalNLLeft := parseBytes(d[0:splitAt], readings)
	cutBegin, cutEnd := terminalNLLeft, splitAt+initialNLRight
	if cutEnd-cutBegin >= lenMinReading {
		if initialNLLeft > noNewline {
			parseBytes(d[cutBegin+1:cutEnd+1], readings)
		} else {
			parseBytes(d[cutBegin:cutEnd+1], readings)
		}
	} else if cutEnd-cutBegin > 0 {
		panic("the ignored reading data is too short")
	}
}

func parseBytes(d []byte, readings chan<- *reading) (initialNL, terminalNL int) {
	if len(d) < lenMinReading {
		panic(fmt.Sprintf("too few bytes: \"%s\"", d))
	}
	initialNL = noNewline
	i := len(d) - 1
	// Ignore anything after the terminal newline in the byte slice.
	for ; i > 0; i-- {
		if d[i] == '\n' {
			terminalNL = i
			i--
			break
		}
	}
	if i == 0 {
		return
	}
	var terminalNameByteIndex int
nextReading:
	// TODO: test if instantiating this as a pointer improves performance.
	parsed := reading{}
	// Tenths
	parsed.temp[2] = d[i] &^ '0'
	i -= 2 // skip the dot
	// Ones
	parsed.temp[1] = d[i] &^ '0'
	i--
	// If a semicolon, return early, the rest is the name.
	if d[i] == ';' {
		parsed.station = d[0:i]
		goto consumeName
	}
	// Either a minus or a number in the tens place.
	if d[i] != '-' {
		parsed.temp[0] = d[i] &^ '0'
		i--
	}
	// Must either be a hyphen-minus or semicolon.
	if d[i] == '-' {
		// It's a hyphen-minus, so the temp is negative.
		parsed.subzero = true
		i--
	}
	// d[i] must be a semicolon at this point.
consumeName:
	terminalNameByteIndex = i
	for ; i > 0; i-- {
		if d[i] == '\n' {
			initialNL = i
			parsed.station = d[i+1 : terminalNameByteIndex]
			readings <- &parsed
			i--
			if initialNL-lenMinReading >= -1 {
				goto nextReading
			}
		}
	}
	if d[i] == '\n' {
		parsed.station = d[i+1 : terminalNameByteIndex]
		readings <- &parsed
		return 0, terminalNL
	}
	if initialNL == noNewline {
		parsed.station = d[i:terminalNameByteIndex]
		readings <- &parsed
		return
	}
	return
}
