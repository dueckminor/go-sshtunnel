# client-hook

The `libsshtunnel-client-hook.dylib` can be used for applications which don't
support http or socks5 proxies.

## Installation

To build the library just call `./scripts/build-client-hook`

There is no installation required. Just copy the `libsshtunnel-client-hook.dylib|so`
library somewhere on your computer and set some environment variables.
### macOS

```bash
export DYLD_INSERT_LIBRARIES=<path to the libsshtunnel-client-hook.dylib|so library>
export DYLD_FORCE_FLAT_NAMESPACE=1
export SSHTUNNEL_PROXY="http://localhost:<port of the http proxy>"
```

### Linux

Not yet supported

## Limitations

- Works only on macOS (It should be possible to add Linux support with low effort)
- Binaries shipped by Apple can not be hooked (or maybe all notarized binaries!?)
- Works only for binaries which are using `getaddrinfo` and `connect` to establish
  a connection

## Todo's

- It should be possible to configure a allow list for the executables which
  shall be hooked
- There should be a list of executables where the environment variables
  should not be inherited to their child processes

## References

I've been inspired by the really old `tsocks` project (https://tsocks.sourceforge.net/). But this project seems to be dead.
And for my use case it's not really sufficient, as I also need DNS forwarding.

There is also a port for macOS (https://github.com/zouguangxian/tsocks),
but it seems that it's not macOS Ventura (13.1) compatible.

Therefor I decided to implement a tiny library which just wraps `getaddrinfo`
and `connect`. Instead of using socks5, I've used and extended the http
proxy in sshtunnel to support not only the method `CONNECT` but also a
method which I have named `RESOLVE`. This method translates the hostname into
a IP address.
