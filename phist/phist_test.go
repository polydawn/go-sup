package phist

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test(t *testing.T) {
	Convey("ShouldSequence should check sequencing", t, func() {
		Convey("A slice which contains the elements in correct sequence should pass", func() {
			So(
				ShouldSequence(
					[]string{"a", "c", "b"},
					"a", "b",
				),
				ShouldEqual,
				"",
			)
		})
		Convey("Elements in an incorrect order fail", func() {
			So(
				ShouldSequence(
					[]string{"b", "a", "c"},
					"a", "b",
				),
				ShouldEqual,
				`Sequence broken: at index 0: "b" occured 1 times, overtaking "a" which is supposed to precede it but only occured 0`,
			)
		})
		Convey("Recurring elements pass as long as they can be pairwise ordered", func() {
			So(
				ShouldSequence(
					[]string{"a", "c", "a", "b", "b"},
					"a", "b",
				),
				ShouldEqual,
				"",
			)
		})
		Convey("Recurring elements fail if later parts of the sequence outnumber their precursors", func() {
			So(
				ShouldSequence(
					[]string{"a", "c", "b", "b", "a"},
					"a", "b",
				),
				ShouldEqual,
				`Sequence broken: at index 3: "b" occured 2 times, overtaking "a" which is supposed to precede it but only occured 1`,
			)
		})
		Convey("Recurring elements fail if sequences start but lack matches", func() {
			So(
				ShouldSequence(
					[]string{"a", "c", "a", "b", "e"},
					"a", "b",
				),
				ShouldEqual,
				`Sequence broken: at end, "a" occured 2 times, and "b" which is supposed to follow it only occured 1`,
			)
		})
		Convey("Lack of relevant elements fails to match", func() {
			So(
				ShouldSequence(
					[]string{"a", "c", "a", "b", "e"},
					"f", "g",
				),
				ShouldEqual,
				`Sequence broken: none of the keywords ever encountered`,
			)
		})
	})
}
