
### Prior excellent discussions of channels and how to think about their semantics:

- https://inconshreveable.com/07-08-2014/principles-of-designing-go-apis-with-channels/
  - In particular, that it's often most correct for functions to *accept* a writable channel as a parameter instead of returning one,
    because this kind of gather pattern means selecting over bulks of these events works fine.
- http://blog.golang.org/pipelines
  - Discussions of patterns for 'Done' channels and cancellation

### various channel use strategies

- Closing a chan is the only way to send an event to $n receivers at once
  - It's also incapable of blocking the closer, which is hugely useful for systemic reliability.
  - but also means you must *return* a chan, which is incompatible with selecting :(
    - maybe a pattern of supervisors (or lighter fold-in'ers) that gathers unblockable close ops into a report stream is useful.  "wastes" a goroutine, but.. reliable.
      - ... it might be useful to combine this with latchery: unblockable close ops where they matter; one (standard) fan out worker that can warn on slow recvrs
- Accepting a chan for gathering is great
  - ... but now if you care about anything but the sheer number of events, the chan's message now needs to contain meaning (as opposed to a close-only signalling chan, where uniqueness yada yada)
    - hello there generics problem
  - you also must keep a list of each chan you've accepted, now, and sync around that (it's not bounded).
    - (repeatr already did this in the knowledge-base system in several places, it appears to be a common pattern)

### compare: x/net/context

- By and large: this is the right direction, absolutely.  But it's not everything.
- Bugs me: The factory methods are kind of helterskelter, and I shouldn't have to implement those for new context types.
  - This is perhaps nitpicky, but I'd really prefer the API strongly guides you towards deadlines (a relative timeout function should be *more* typing, because it's almost always less correct).
- The "Value" bag thing is something I'd really rather remained a separate concept.
- Overall, this is a system for doing cancellations, and that's fine, but it doesn't have much to say about collecting errors on the way up again, and that is a thing I'm interested in regularizing.
  - There's sense to this: x/net/context is a parameter you pass down, and that's it, and you still use control flow for returns... and by doing so, you never get stuck in the generics desire.
