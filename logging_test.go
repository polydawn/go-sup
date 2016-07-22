package sup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLogging(t *testing.T) {
	Convey("WritNames should dtrt", t, func() {
		wn := &WritName{}

		Convey("Empty writnames serialize specially", func() {
			So(wn.String(), ShouldResemble, "[root]")
			So(wn.Coda(), ShouldResemble, "[root]")
		})

		Convey("Appending a segment works", func() {
			wn1 := wn.New("sys")
			So(wn1.String(), ShouldResemble, "sys")
			So(wn1.Coda(), ShouldResemble, "sys")

			Convey("Parent writnames was not modified", func() {
				So(wn.String(), ShouldResemble, "[root]")
				So(wn.Coda(), ShouldResemble, "[root]")
			})

			Convey("Chains of segment work", func() {
				wn2 := wn.New("laser").New("bank1")
				So(wn2.String(), ShouldResemble, "laser.bank1")
				So(wn2.Coda(), ShouldResemble, "bank1")
			})
		})
	})
}
