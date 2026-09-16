# tissla-wallpaper

A simple, dependency-free wayland wallpaper daemon and CLI. 
The wayland-client is hand-written from the wayland.xml protocol, opting not to use a code generator.

Compared to more memory-efficient languages, the Go runtime tends to retain unused heap memory for future allocations. However, if the memory remains unused, Go's runtime scavenger will eventually return it to the operating system, often within minutes, making the long-term difference in memory usage much smaller. 

This software is part of my personal installation scripts, that can be found in my [Dotfiles](https://github.com/tissla/tissla-dotfiles).

## Future?
Custom transition effects.
I also plan to add support for animated wallpapers, though plainly for the challenge of it, as I don't use them myself.
