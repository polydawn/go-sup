SUP
===

Supervisors for Go(lang).



Why?
----

Collecting goroutines is important.
It's simply systematically *sane*.
Letting goroutines spin off into the distance with no supervision is extremely likely to represent either a resource leak,
a missing error handle path, or an outright logic bug.

There's been a litany [1][2] of blogs about this recently, if anyone doesn't consider this self-evident.

- [1] https://dave.cheney.net/2016/12/22/never-start-a-goroutine-without-knowing-how-it-will-stop
- [2] https://rakyll.org/leakingctx/

In a dream world, we might have Erlang-style supervisor trees so this can all be done with less boilerplate.
It's almost even possible to do this as a library in go!

And that's what go-sup is.  Supervisor trees as a library.



What this doesn't solve
-----------------------

Quitting is still "cooperative" -- code must be well behaved, and respond to the quit in a reasonable time.

In most situations, well-behaved code is not terribly complicated to write.
However, blocking IO often still presents [a bit of an issue](https://news.ycombinator.com/item?id=13332185).
Any supervisor of a goroutine that may be IO-blocked may itself be indefinitely stuck, and so on up-tree.
Typically this can at least be salved by using timeouts to minimize the worst case for block times.

go-sup will issue warnings messages (the function for this is configurable -- the default is printing to stderr)
for tasks that do not return within a reasonable time (2 seconds).
