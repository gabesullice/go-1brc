package report

import (
	"bytes"
	"hash/fnv"
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

func Test_parseFileLeftRight(t *testing.T) {
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
			"Aix-en-Provence;4.2\nx;6.9\n", // shortest valid byte slice
			&record{name: []byte("Aix-en-Provence"), min: 42, max: 42, sum: 42},
			&record{name: []byte("x"), min: 69, max: 69, sum: 69},
		),
		testCase(
			"bar;1.0\nfoo;2.0\nfoo;2.0\nfoo;2.0\nfoo;2.0\nfoo;2.0\nbar;1.0\nbar;1.0\nfoo;2.0\n", // shortest valid byte slice
			&record{name: []byte("bar"), min: 10, max: 10, sum: 10 + 10 + 10},
			&record{name: []byte("foo"), min: 20, max: 20, sum: 20 + 20 + 20 + 20 + 20 + 20},
		),
	}
	for _, tc := range testCases {
		t.Run(string(tc.input), func(t *testing.T) {
			readings := &tree{}
			parseFileLeftRight(bytes.NewReader(tc.input), 0, len(tc.input), readings)
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
			"\nAix-en-Provence;-24.5\n",
			&record{name: []byte("Aix-en-Provence"), min: -245, max: -245, sum: -245},
		),
		testCase(
			"ignore;-24.5\nAix-en-Provence;-24.5\n",
			&record{name: []byte("Aix-en-Provence"), min: -245, max: -245, sum: -245},
		),
		testCase(
			"\nAix-en-Provence;-24.5\nignore;-",
			&record{name: []byte("Aix-en-Provence"), min: -245, max: -245, sum: -245},
		),
		testCase(
			"ignore;-24.5\nAix-en-Provence;-24.5\nignore;-",
			&record{name: []byte("Aix-en-Provence"), min: -245, max: -245, sum: -245},
		),
		testCase(
			"Aix-en-Provence;-24.5\nDenver;0.0\n",
			&record{name: []byte("Denver"), min: 0, max: 0, sum: 0},
		),
		testCase(
			"\nAix-en-Provence;-24.5\nDenver;0.0\n",
			&record{name: []byte("Aix-en-Provence"), min: -245, max: -245, sum: -245},
			&record{name: []byte("Denver"), min: 0, max: 0, sum: 0},
		),
		testCase(
			"ignore;-24.5\nAix-en-Provence;-24.5\nDenver;0.0\n",
			&record{name: []byte("Aix-en-Provence"), min: -245, max: -245, sum: -245},
			&record{name: []byte("Denver"), min: 0, max: 0, sum: 0},
		),
		testCase(
			"\nAix-en-Provence;-24.5\nDenver;0.0\nignore;-",
			&record{name: []byte("Aix-en-Provence"), min: -245, max: -245, sum: -245},
			&record{name: []byte("Denver"), min: 0, max: 0, sum: 0},
		),
		testCase(
			"ignore;-24.5\nAix-en-Provence;-24.5\nDenver;0.0\nignore;-",
			&record{name: []byte("Aix-en-Provence"), min: -245, max: -245, sum: -245},
			&record{name: []byte("Denver"), min: 0, max: 0, sum: 0},
		),
	}
	for _, tc := range testCases {
		t.Run(string(tc.input), func(t *testing.T) {
			var initialNL, terminalNL int
			readings := &tree{}
			initialNL, terminalNL = parseBytes(tc.input, readings)
			assertReadings(t, tc.expect, readings)
			expectInitialNL := bytes.IndexByte(tc.input, '\n')
			expectTerminalNL := bytes.LastIndexByte(tc.input, '\n')
			if terminalNL != expectTerminalNL {
				t.Errorf("expected terminal newline at: %d; got %d", expectTerminalNL, terminalNL)
			}
			if expectInitialNL == expectTerminalNL {
				if initialNL != noNewline {
					t.Errorf("expected initial newline at: %d; got %d", noNewline, initialNL)
				}
			} else {
				if initialNL != expectInitialNL {
					t.Errorf("expected initial newline at: %d; got %d", expectInitialNL, initialNL)
				}
			}
		})
	}
}

func assertReadings(t *testing.T, expected []*record, records *tree) {
	t.Helper()
	actual := records.flatten()
	if len(actual) != len(expected) {
		t.Errorf("expected %d records; got %d", len(expected), len(actual))
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
		t.Errorf("expected min to be: %d; got: %d", expect.min, actual.min)
	}
	if actual.max != expect.max {
		t.Errorf("expected max to be: %d; got: %d", expect.max, actual.max)
	}
	if actual.sum != expect.sum {
		t.Errorf("expected sum to be: %d; got: %d", expect.sum, actual.sum)
	}
}

func Test_fnv(t *testing.T) {
	expect := fnv.New64()
	expect.Write([]byte("x"))
	hash := fnvOffsetBasis
	hash *= fnvPrime
	hash ^= uint64('x')
	if hash != expect.Sum64() {
		t.FailNow()
	}
	expect = fnv.New64()
	expect.Write([]byte("xyz"))
	hash = fnvOffsetBasis
	hash *= fnvPrime
	hash ^= uint64('x')
	hash *= fnvPrime
	hash ^= uint64('y')
	hash *= fnvPrime
	hash ^= uint64('z')
	if hash != expect.Sum64() {
		t.FailNow()
	}
}
