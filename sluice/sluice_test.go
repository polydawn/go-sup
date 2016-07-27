package sluice

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test(t *testing.T) {
	Convey("Sluice can...", t, func() {
		gondola := New()

		Convey("pump values", func() {
			gondola.Push("x")
			gondola.Push("y")
			gondola.Push("z")
			So(<-gondola.Next(), ShouldEqual, "x")
			So(<-gondola.Next(), ShouldEqual, "y")
			So(<-gondola.Next(), ShouldEqual, "z")
		})

		Convey("block when empty", func() {
			var answered bool
			select {
			case <-gondola.Next():
				answered = true
			default:
				answered = false
			}
			So(answered, ShouldEqual, false)

			Convey("answers even dropped channels", func() {
				secondReq := gondola.Next()
				gondola.Push("1")
				// we still don't expect an answer,
				//  because the "1" routed to the channel in the prev test.
				select {
				case <-secondReq:
					answered = true
				default:
					answered = false
				}
				So(answered, ShouldEqual, false)

				Convey("definitely answers eventually", func() {
					gondola.Push("2")
					So(<-secondReq, ShouldEqual, "2")
				})
			})
		})
	})
}
