package phist

import "fmt"

/*

	Any slice which contains the elements in correct sequence passes:

		So(
			[]{"a", "c", "b"},
			ShouldSequence,
			"a", "b",
		) => true

	Elements in an incorrect order fail:

		So(
			[]{"b", "a", "c"},
			ShouldSequence,
			"a", "b",
		) => false

	When elements recur, as long as they could be pairwise ordered, they pass:

		So(
			[]{"a", "c", "a", "b", "b"},
			ShouldSequence,
			"a", "b",
		) => true

		So(
			[]{"a", "c", "b", "b", "a"},
			ShouldSequence,
			"a", "b",
		) => false

	When elements recur, they must have the latter half at least as often
	as the first half:

		So(
			[]{"a", "c", "a", "b", "e"},
			ShouldSequence,
			"a", "b",
		) => false

*/
func ShouldSequence(actual interface{}, expected ...interface{}) string {
	// args parsery
	sequence, ok := actual.([]string)
	if !ok {
		return "You must provide a string slice as the first argument to this assertion."
	}
	var keywords []string
	var counts []int
	switch len(expected) {
	case 0, 1:
		return "You must provide at least two parameters as expectations to this assertion."
	default:
		for _, v := range expected {
			keyword, ok := v.(string)
			if !ok {
				return fmt.Sprintf("You must provide strings as expected values, not %T.", v)
			}
			keywords = append(keywords, keyword)
			counts = append(counts, 0)
		}
	}

	// run checks against the seq
	for i, val := range sequence {
		for j, kw := range keywords {
			if val == kw {
				counts[j]++
				if j == 0 { // first event can always happen again
					continue
				}
				if counts[j-1] < counts[j] {
					return fmt.Sprintf("Sequence broken: at index %d: %q occured %d times, overtaking %q which is supposed to precede it but only occured %d",
						i,
						keywords[j], counts[j],
						keywords[j-1], counts[j-1],
					)
				}
				continue
			}
		}
	}
	// at the end, check that there are no more unmatched sequence starts
	for i := 1; i < len(counts); i++ {
		if counts[i-1] > counts[i] {
			return fmt.Sprintf("Sequence broken: at end, %q occured %d times, and %q which is supposed to follow it only occured %d",
				keywords[i-1], counts[i-1],
				keywords[i], counts[i],
			)
		}
	}
	// also, just not existing is not a valid match
	if counts[0] == 0 {
		return "Sequence broken: none of the keywords ever encountered"
	}

	return ""
}

/*
	Checks that an element occurs $n times:

		So(
			[]{"a", "c", "b", "c"},
			ShouldOccurWithFrequency,
			"c", 2,
		) => true
*/
func ShouldOccurWithFrequency(actual interface{}, expected ...interface{}) string {
	return "todo"
}

/*
	Checks that all instances of an element occur before the first incident
	of another:

		So(
			[]{"a", "a", "a", "b"},
			ShouldAllPrecede,
			"a", "b",
		) => true

		So(
			[]{"a", "a", "b", "a"},
			ShouldAllPrecede,
			"a", "b",
		) => false
*/
func ShouldAllPrecede(actual interface{}, expected ...interface{}) string {
	return "todo"
}

/*
	Same semantics as `ShouldAllPrecede`, in the opposite direction:

		So(
			[]{"a", "b", "b", "b"},
			ShouldAllFollow,
			"b", "a",
		) => true
*/
func ShouldAllFollow(actual interface{}, expected ...interface{}) string {
	return "todo"
}
