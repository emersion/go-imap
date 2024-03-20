# go-sasl

[![godocs.io](https://godocs.io/github.com/emersion/go-sasl?status.svg)](https://godocs.io/github.com/emersion/go-sasl)
[![Build Status](https://travis-ci.org/emersion/go-sasl.svg?branch=master)](https://travis-ci.org/emersion/go-sasl)

A [SASL](https://tools.ietf.org/html/rfc4422) library written in Go.

Implemented mechanisms:

* [ANONYMOUS](https://tools.ietf.org/html/rfc4505)
* [EXTERNAL](https://tools.ietf.org/html/rfc4422#appendix-A)
* [LOGIN](https://tools.ietf.org/html/draft-murchison-sasl-login-00) (obsolete, use PLAIN instead)
* [PLAIN](https://tools.ietf.org/html/rfc4616)
* [OAUTHBEARER](https://tools.ietf.org/html/rfc7628)

## License

MIT
