package report

import (
	"bytes"
	"hash/fnv"
	"os"
	"testing"
)

type test struct {
	input  []byte
	expect []*record
}

func testCase(input string, expect ...*record) (tc test) {
	tc.input = []byte(input)
	tc.expect = expect
	return
}

func Test_bytesAfterLastByte(t *testing.T) {
	haystack := []byte("Aix-en-Provence;4.2\nx;6.9")
	actual := bytesAfterLastByte(bytes.NewReader(haystack), len(haystack), '\n')
	expect := 5
	if expect != actual {
		t.Errorf("expected: %d; got: %d", expect, actual)
	}
}

func Test_parseLargeFile(t *testing.T) {
	f, err := os.Open("./testdata/measurements-10e5.txt")
	if err != nil {
		panic(err)
	}
	readings := parseFile(f)
	records := readings.flatten()
	if len(records) != 413 {
		t.Errorf("expected %d records; got: %d", 413, len(records))
	}
}

func Test_parseFile(t *testing.T) {
	f, err := os.Open("./testdata/measurements-10e1.txt")
	if err != nil {
		panic(err)
	}
	readings := parseFile(f)
	records := readings.flatten()
	if len(records) != 90 {
		t.Errorf("expected %d records; got: %d", 90, len(records))
	}
}

func Test_parseComplete(t *testing.T) {
	testCases := []test{
		testCase(
			"x;4.2\nx;6.9\n",
			&record{name: []byte("x"), min: 42, max: 69, sum: 42 + 69},
		),
		testCase(
			"x;4.2\nx;4.2\nx;6.9\n",
			&record{name: []byte("x"), min: 42, max: 69, sum: 42 + 42 + 69},
		),
		testCase(
			"x;4.2\nx;42.0\nx;4.2\nx;6.9\n",
			&record{name: []byte("x"), min: 42, max: 420, sum: 42 + 420 + 42 + 69},
		),
		testCase(
			"Aix-en-Provence;4.2\nx;6.9\n",
			&record{name: []byte("Aix-en-Provence"), min: 42, max: 42, sum: 42},
			&record{name: []byte("x"), min: 69, max: 69, sum: 69},
		),
		testCase(
			"aaa;1.0\nbbb;1.0\nccc;1.0\nddd;1.0\neee;1.0\nfff;1.0\nggg;1.0\nhhh;1.0\niii;1.0\niii;1.0\n",
			&record{name: []byte("aaa"), min: 10, max: 10, sum: 10},
			&record{name: []byte("bbb"), min: 10, max: 10, sum: 10},
			&record{name: []byte("ccc"), min: 10, max: 10, sum: 10},
			&record{name: []byte("ddd"), min: 10, max: 10, sum: 10},
			&record{name: []byte("eee"), min: 10, max: 10, sum: 10},
			&record{name: []byte("fff"), min: 10, max: 10, sum: 10},
			&record{name: []byte("ggg"), min: 10, max: 10, sum: 10},
			&record{name: []byte("hhh"), min: 10, max: 10, sum: 10},
			&record{name: []byte("iii"), min: 10, max: 10, sum: 20},
		),
		testCase(
			"Jerusalem;25.8\nLa Paz;26.1\nDenpasar;16.2\nNew Delhi;20.7\nMandalay;21.2\nOdesa;17.1\nErbil;28.7\nSan Francisco;18.9\nAthens;21.4\nBangkok;26.5\n",
			&record{name: []byte("Athens"), min: 214, max: 214, sum: 214},
			&record{name: []byte("Bangkok"), min: 265, max: 265, sum: 265},
			&record{name: []byte("Denpasar"), min: 162, max: 162, sum: 162},
			&record{name: []byte("Erbil"), min: 287, max: 287, sum: 287},
			&record{name: []byte("Jerusalem"), min: 258, max: 258, sum: 258},
			&record{name: []byte("La Paz"), min: 261, max: 261, sum: 261},
			&record{name: []byte("Mandalay"), min: 212, max: 212, sum: 212},
			&record{name: []byte("New Delhi"), min: 207, max: 207, sum: 207},
			&record{name: []byte("Odesa"), min: 171, max: 171, sum: 171},
			&record{name: []byte("San Francisco"), min: 189, max: 189, sum: 189},
		),
	}
	for _, tc := range testCases {
		t.Run(string(tc.input), func(t *testing.T) {
			readings := newTree()
			buf := make([]byte, 0, maxReadLength)
			parseComplete(bytes.NewReader(tc.input), len(tc.input), buf, readings)
			assertReadings(t, tc.expect, readings)
		})
	}
}

func Test_parseBytes(t *testing.T) {
	testCases := []test{
		testCase(
			"x;4.2\n", // shortest valid byte slice
			&record{name: []byte("x"), min: 42, max: 42, sum: 42},
		),
		testCase(
			"Aix-en-Provence;0.0\n",
			&record{name: []byte("Aix-en-Provence"), min: 0, max: 0, sum: 0},
		),
		testCase(
			"Aix-en-Provence;3.5\n",
			&record{name: []byte("Aix-en-Provence"), min: 35, max: 35, sum: 35},
		),
		testCase(
			"Aix-en-Provence;30.5\n",
			&record{name: []byte("Aix-en-Provence"), min: 305, max: 305, sum: 305},
		),
		testCase(
			"Aix-en-Provence;24.5\n",
			&record{name: []byte("Aix-en-Provence"), min: 245, max: 245, sum: 245},
		),
		testCase(
			"Aix-en-Provence;-0.0\n",
			&record{name: []byte("Aix-en-Provence"), min: 0, max: 0, sum: 0},
		),
		testCase(
			"Aix-en-Provence;-3.5\n",
			&record{name: []byte("Aix-en-Provence"), min: -35, max: -35, sum: -35},
		),
		testCase(
			"Aix-en-Provence;-30.5\n",
			&record{name: []byte("Aix-en-Provence"), min: -305, max: -305, sum: -305},
		),
		testCase(
			"Aix-en-Provence;-24.5\n",
			&record{name: []byte("Aix-en-Provence"), min: -245, max: -245, sum: -245},
		),
		testCase(
			"Aix-en-Provence;-24.5\nDenver;0.0\n",
			&record{name: []byte("Aix-en-Provence"), min: -245, max: -245, sum: -245},
			&record{name: []byte("Denver"), min: 0, max: 0, sum: 0},
		),
	}
	for _, tc := range testCases {
		t.Run(string(tc.input), func(t *testing.T) {
			readings := newTree()
			terminalNL := parseBytes(tc.input, readings)
			assertReadings(t, tc.expect, readings)
			expectTerminalNL := bytes.LastIndexByte(tc.input, '\n')
			if terminalNL != expectTerminalNL {
				t.Errorf("expected terminal newline at: %d; got %d", expectTerminalNL, terminalNL)
			}
		})
	}
}

func assertReadings(t *testing.T, expected []*record, records *tree) {
	t.Helper()
	actual := records.flatten()
	if len(actual) != len(expected) {
		t.Errorf("expected %d records; got %d", len(expected), len(actual))
		return
	}
	for i, expect := range expected {
		assertReading(t, expect, actual[i])
	}
}

func assertReading(t *testing.T, expect *record, actual *record) {
	t.Helper()
	if !bytes.Equal(expect.name, actual.name) {
		t.Errorf("expected: %s; got: %s", expect.name, actual.name)
	}
	if actual.min != expect.min {
		t.Errorf("expected %s min to be: %d; got: %d", expect.name, expect.min, actual.min)
	}
	if actual.max != expect.max {
		t.Errorf("expected %s max to be: %d; got: %d", expect.name, expect.max, actual.max)
	}
	if actual.sum != expect.sum {
		t.Errorf("expected %s sum to be: %d; got: %d", expect.name, expect.sum, actual.sum)
	}
}

func Test_fnv(t *testing.T) {
	expect := fnv.New32a()
	expect.Write([]byte("x"))
	hash := fnvOffsetBasis
	hash ^= uint32('x')
	hash *= fnvPrime
	if hash != expect.Sum32() {
		t.FailNow()
	}
	expect = fnv.New32a()
	expect.Write([]byte("xyz"))
	hash = fnvOffsetBasis
	hash ^= uint32('x')
	hash *= fnvPrime
	hash ^= uint32('y')
	hash *= fnvPrime
	hash ^= uint32('z')
	hash *= fnvPrime
	if hash != expect.Sum32() {
		t.FailNow()
	}
}
