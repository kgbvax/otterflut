# otterflut
A pixelflut server in golang,
maybe a fast one and
maybe one with "pluggable rules".

# Goals
* In golang (just because)
* Fast, consume a full gigabit on an ARM (Odroid XU4)
* With pluggable "rules" to make it more interesting...
* Maybe: Insanely fast, consume 10Gbit/sec on an ARM board like MACCHIATObin

# Status
* Works on Mac,Linux, requires SDL2
* parsing "PX messages" and settings pixels currently works with well over 1Gbit/sec on an Odroid XU4
* reading from network is slow as molasses

# Why?
At Easterhegg18 we (again) has some fun with Pixelflut and I asked myself whether it would be more interesting if we had slightly different rules. For that a server that implements these rules would be needed.
(related [Trööt](https://chaos.social/@kgbvax/99778010521874836))

Also @larsm laptop failed because of overheating (at 1GBit) at times. We also discussed ways to make pixelflut fast again and OpenGl was mentioned.
My theory is that "settings pixels" by CPU would be the fastes way to do things since all GPU interaction requires simply a too many software layers.
This is the proof-of-concept of this.
