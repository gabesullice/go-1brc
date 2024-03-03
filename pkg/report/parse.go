package report

import "fmt"

type reading struct {
	station     []byte
	temperature int64
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
	var semicolonIndex int
nextReading:
	// TODO: test if instantiating this as a pointer improves performance.
	parsed := reading{}
	// Tenths
	temp := d[i] &^ '0'
	i -= 2 // skip the dot
	// Ones
	temp += d[i] &^ '0' * 10
	i--
	// If a semicolon, return early, the rest is the name.
	if d[i] == ';' {
		parsed.station = d[0:i]
		parsed.temperature = int64(temp)
		goto consumeName
	}
	// Either a minus or a number in the tens place.
	if d[i] != '-' {
		parsed.temperature = int64(d[i]&^'0')*100 + int64(temp)
		i--
	} else {
		parsed.temperature = int64(temp)
	}
	// Must either be a hyphen-minus or semicolon.
	if d[i] == '-' {
		// It's a hyphen-minus, so the temp is negative.
		parsed.temperature *= -1
		i--
	}
consumeName:
	// d[i] must be a semicolon at this point.
	semicolonIndex = i
	i--
	for ; i > 0; i-- {
		if d[i] == '\n' {
			initialNL = i
			parsed.station = d[i+1 : semicolonIndex]
			readings <- &parsed
			i--
			if initialNL-lenMinReading >= -1 {
				goto nextReading
			}
		}
	}
	if d[i] == '\n' {
		parsed.station = d[i+1 : semicolonIndex]
		readings <- &parsed
		return 0, terminalNL
	}
	if initialNL == noNewline {
		parsed.station = d[i:semicolonIndex]
		readings <- &parsed
		return
	}
	return
}
